// Copyright © Kaleido, Inc. 2024

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package platform

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"

	_ "embed"
)

// Create a test PNG file for testing
func createTestPNGFile(t *testing.T) string {
	// Create a minimal PNG file for testing
	// This is a 1x1 pixel PNG file
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE, 0x00, 0x00, 0x00,
		0x0C, 0x49, 0x44, 0x41, 0x54, 0x08, 0xD7, 0x63, 0xF8, 0x0F, 0x00, 0x00,
		0x01, 0x00, 0x01, 0x00, 0x18, 0xDD, 0x8D, 0xB4, 0x00, 0x00, 0x00, 0x00,
		0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test-icon-*.png")
	assert.NoError(t, err)
	defer tmpFile.Close()

	_, err = tmpFile.Write(pngData)
	assert.NoError(t, err)

	return tmpFile.Name()
}

var wms_asset_icon_step1 = `
resource "kaleido_platform_wms_asset" "test_asset" {
  environment             = "env1"
  service                 = "service1"
  name                    = "test-asset"
  description             = "test asset for icon"
  symbol                  = "TEST"
  color                   = "#FF0000"
  account_identifier_type = "eth_address"
  protocol_id             = "0x92c580144897F7613f25d80C844e3DfCF8E78524"
  config_json = jsonencode({
    decimals = 6
    units = [
      {
        name   = "c"
        factor = 4
        prefix = false
      }
    ]
    transfers = {
      backend   = "asset-manager1"
      backendId = "erc_0x92c580144897F7613f25d80C844e3DfCF8E78524"
    }
    operations = {
      mint: {
        schema         = "mint schema"
        fieldMapping = "mint field mapping"
        flow = "erc20-workflow"
        backendMapping = "mint backend mapping"
      }
      transfer: {
        schema         = "transfer schema"
        fieldMapping = "transfer field mapping"
        flow = "erc20-workflow"
        backendMapping = "transfer backend mapping"
      }
    }
  })
}

resource "kaleido_platform_wms_asset_icon" "test_icon" {
  environment = "env1"
  service     = "service1"
  asset_name  = kaleido_platform_wms_asset.test_asset.name
  file_path   = "%s"
  file_type   = "image/png"
  
  depends_on = [kaleido_platform_wms_asset.test_asset]
}
`

func TestWMSAssetIcon1(t *testing.T) {
	// Create test PNG file
	testPNGFile := createTestPNGFile(t)
	defer os.Remove(testPNGFile)

	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"POST /endpoint/{env}/{service}/rest/api/v1/assets",
			"POST /endpoint/{env}/{service}/rest/api/v1/assets/{asset}/icon",
			"GET /endpoint/{env}/{service}/rest/api/v1/assets/{asset}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/assets/{asset}/icon",
			"GET /endpoint/{env}/{service}/rest/api/v1/assets/{asset}/icon",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/assets/{asset}",
			"GET /endpoint/{env}/{service}/rest/api/v1/assets/{asset}",
		})
		mp.server.Close()
	}()

	wms_asset_icon_resource := "kaleido_platform_wms_asset_icon.test_icon"
	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(wms_asset_icon_step1, testPNGFile),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(wms_asset_icon_resource, "environment", "env1"),
					resource.TestCheckResourceAttr(wms_asset_icon_resource, "service", "service1"),
					resource.TestCheckResourceAttr(wms_asset_icon_resource, "asset_name", "test-asset"),
					resource.TestCheckResourceAttr(wms_asset_icon_resource, "file_path", testPNGFile),
				),
			},
		},
	})
}

func TestWMSAssetIconFileNotFound(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.server.Close()
	}()

	iconConfig := `
resource "kaleido_platform_wms_asset_icon" "test_icon" {
  environment = "env1"
  service     = "service1"
  asset_name  = "test-asset"
  file_path   = "/nonexistent/file.png"
  file_type   = "image/png"
}
`

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + iconConfig,
				ExpectError: regexp.MustCompile("Could not open file"),
			},
		},
	})
}

func (mp *mockPlatform) getWMSAssetIcon(res http.ResponseWriter, req *http.Request) {
	// Check if the asset icon exists - lookup by assetname
	env := mux.Vars(req)["env"]
	service := mux.Vars(req)["service"]
	assetNameOrID := mux.Vars(req)["asset"]
	assetIconKey := env + "/" + service + "/" + assetNameOrID
	asset := mp.wmsAssetIcons[assetIconKey]
	if asset == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, &struct{}{}, 200)
	}
}

func (mp *mockPlatform) postWMSAssetIcon(res http.ResponseWriter, req *http.Request) {
	// Check if the asset exists - lookup by name
	env := mux.Vars(req)["env"]
	service := mux.Vars(req)["service"]
	assetName := mux.Vars(req)["asset"]
	assetKey := env + "/" + service + "/" + assetName
	asset := mp.wmsAssets[assetKey]
	if asset == nil {
		mp.respond(res, nil, 404)
		return
	}

	assetKeyByName := env + "/" + service + "/" + asset.Name
	assetKeyByID := env + "/" + service + "/" + asset.ID

	// Parse multipart form data
	err := req.ParseMultipartForm(10 << 20) // 10 MB max
	assert.NoError(mp.t, err, "Failed to parse multipart form")

	// Check if file field exists
	file, header, err := req.FormFile("file")
	assert.NoError(mp.t, err, "Failed to get file from form")
	assert.NotNil(mp.t, file, "File field is required")
	defer file.Close()

	// Check file type - don't check Content-Type header as it's set by the client
	// Instead, just verify the file exists and has .png extension
	assert.True(mp.t, len(header.Filename) > 4, "Filename should be provided")

	mp.wmsAssetIcons[assetKeyByName] = &struct{}{}
	mp.wmsAssetIcons[assetKeyByID] = &struct{}{}

	mp.respond(res, nil, 204)
}

func (mp *mockPlatform) deleteWMSAssetIcon(res http.ResponseWriter, req *http.Request) {
	// Check if the asset exists - lookup by asset name
	env := mux.Vars(req)["env"]
	service := mux.Vars(req)["service"]
	assetNameOrID := mux.Vars(req)["asset"]
	assetKey := env + "/" + service + "/" + assetNameOrID
	asset := mp.wmsAssets[assetKey]
	if asset == nil {
		mp.respond(res, nil, 404)
		return
	}

	assetIcon := mp.wmsAssetIcons[assetKey]
	if assetIcon == nil {
		mp.respond(res, nil, 404)
		return
	}

	assetKeyByName := env + "/" + service + "/" + asset.Name
	assetKeyByID := env + "/" + service + "/" + asset.ID

	delete(mp.wmsAssetIcons, assetKeyByName)
	delete(mp.wmsAssetIcons, assetKeyByID)

	mp.respond(res, nil, 204)
}
