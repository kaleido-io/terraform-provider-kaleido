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
	"context"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

type ResourceCommon struct {
	ID      string    `json:"id"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

func withCommon(fields map[string]*schema.Schema) map[string]*schema.Schema {
	fields["id"] = &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	}
	fields["created"] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	fields["updated"] = &schema.Schema{
		Type:     schema.TypeString,
		Computed: true,
	}
	return fields
}

func apiRequest(d *resource.ResourceData) (*resty.Request, context.CancelFunc) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), 120*time.Second)
	return resty.New().
			SetBaseURL(d.ConnInfo()["api"]).
			OnBeforeRequest(func(c *resty.Client, r *resty.Request) error {
				tflog.Infof()
				return nil
			}).
			R().
			SetBasicAuth(d.ConnInfo()["username"], d.ConnInfo()["password"]).
			SetContext(ctx),
		cancelCtx
}

func apiPostResource(d *resource.ResourceData, platformPath string, body interface{}) error {
	d.ConnInfo()
}
