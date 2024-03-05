// Copyright © Kaleido, Inc. 2018, 2024

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package kaleidobase

import (
	"fmt"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

type ProviderData struct {
	BaaS     *kaleido.KaleidoClient
	Platform *resty.Client
}

type ProviderModel struct {
	API              types.String `tfsdk:"api"`
	APIKey           types.String `tfsdk:"api_key"`
	PlatformAPI      types.String `tfsdk:"platform_api"`
	PlatformUsername types.String `tfsdk:"platform_username"`
	PlatformPassword types.String `tfsdk:"platform_password"`
}

func ConfigureProviderData(providerData any, diagnostics diag.Diagnostics) *ProviderData {
	kaleidoProviderData, ok := providerData.(*ProviderData)
	if !ok {
		diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected %T, got: %T. Please report this issue to the provider developers.", kaleidoProviderData, providerData),
		)
		return nil
	}
	return kaleidoProviderData
}

func NewProviderData(conf *ProviderModel) *ProviderData {

	baasAPI := conf.API.ValueString()
	if baasAPI == "" {
		baasAPI = os.Getenv("KALEIDO_API")
	}
	baasAPIKey := conf.API.ValueString()
	if baasAPIKey == "" {
		baasAPIKey = os.Getenv("KALEIDO_API_KEY")
	}
	baas := kaleido.NewClient(baasAPI, baasAPIKey)

	platformAPI := conf.PlatformAPI.ValueString()
	if platformAPI == "" {
		platformAPI = os.Getenv("KALEIDO_PLATFORM_API")
	}
	platformUsername := conf.PlatformUsername.ValueString()
	if platformUsername == "" {
		platformUsername = os.Getenv("KALEIDO_USERNAME")
	}
	platformPassword := conf.PlatformPassword.ValueString()
	if platformPassword == "" {
		platformPassword = os.Getenv("KALEIDO_PASSWORD")
	}
	platform := resty.New().
		SetBaseURL(platformAPI).
		SetBasicAuth(platformUsername, platformPassword)

	return &ProviderData{
		BaaS:     &baas,
		Platform: platform,
	}
}
