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
	"context"
	"crypto/tls"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
)

const version = "v1.1.0"

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

func ConfigureProviderData(providerData any, diagnostics *diag.Diagnostics) *ProviderData {
	if providerData == nil {
		// This oddly happen where Configure is called on resources BEFORE Configure
		// has been called on the provider. We have to handle it, otherwise we don't
		// let things progress to the point that the Configure is called on the plugin.
		// We can a subsequent call for each resource after that.
		return nil
	}
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

func NewProviderData(logCtx context.Context, conf *ProviderModel) *ProviderData {

	baasAPI := conf.API.ValueString()
	if baasAPI == "" {
		baasAPI = os.Getenv("KALEIDO_API")
	}
	baasAPIKey := conf.APIKey.ValueString()
	if baasAPIKey == "" {
		baasAPIKey = os.Getenv("KALEIDO_API_KEY")
	}
	r := resty.New().
		SetTransport(http.DefaultTransport).
		SetBaseURL(baasAPI).
		SetAuthToken(baasAPIKey).
		SetHeader("User-Agent", fmt.Sprintf("Terraform / %s (BaaS)", version))
	AddRestyLogging(logCtx, r)
	baas := &kaleido.KaleidoClient{Client: r}

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

	// mostly the default settings, barring less conns to avoid concurrency limits w/in the Platform
	platformHttp := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          5,
		MaxConnsPerHost:       5,
		MaxIdleConnsPerHost:   5,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	platform := resty.New().
		SetTransport(platformHttp).
		SetHeader("User-Agent", fmt.Sprintf("Terraform / %s (Platform)", version)).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			if err != nil {
				return false
			}
			if r.StatusCode() == http.StatusTooManyRequests {
				return true
			}
			return false
		}).
		SetRetryCount(5).
		SetRetryAfter(func(c *resty.Client, r *resty.Response) (time.Duration, error) {
			tflog.Debug(logCtx, fmt.Sprintf("retryAfter: %s", r.Header().Get("Retry-After")))
			retryAfter, err := strconv.ParseFloat(r.Header().Get("Retry-After"), 64)
			if err != nil {
				return 1 * time.Second, err
			}
			jitter := time.Duration(rand.NormFloat64() * float64(5*time.Second)) // up to 5 second random jitter to account for 5 concurrent connections all retrying up to 5 times
			return time.Duration(retryAfter*float64(time.Second)) + jitter, nil
		}).
		SetBaseURL(platformAPI)
	if platformUsername != "" && platformPassword != "" {
		platform = platform.SetBasicAuth(platformUsername, platformPassword)
	}
	AddRestyLogging(logCtx, platform)

	if os.Getenv("KALEIDO_PLATFORM_INSECURE") == "true" {
		platform.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}

	return &ProviderData{
		BaaS:     baas,
		Platform: platform,
	}
}

func AddRestyLogging(ctx context.Context, rc *resty.Client) {
	rc.OnBeforeRequest(func(c *resty.Client, r *resty.Request) error {
		tflog.Info(ctx, fmt.Sprintf("--> %s %s", r.Method, r.URL))
		return nil
	})
	rc.OnAfterResponse(func(c *resty.Client, r *resty.Response) error {
		tflog.Info(ctx, fmt.Sprintf("<-- %s %s [%d]", r.Request.Method, r.Request.URL, r.StatusCode()))
		return nil
	})
}
