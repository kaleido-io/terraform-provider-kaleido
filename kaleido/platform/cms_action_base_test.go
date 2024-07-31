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
	"encoding/json"
	"net/http"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func (mp *mockPlatform) getCMSAction(res http.ResponseWriter, req *http.Request) {
	obj := mp.cmsActions[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["action"]]
	if obj == nil {
		mp.respond(res, nil, 404)
	} else {
		mp.respond(res, obj, 200)
		// Next time we'll complete the build
		a := obj.OutputBase()
		a.Status = "succeeded"
	}
}

func (mp *mockPlatform) postCMSAction(res http.ResponseWriter, req *http.Request) {
	var base *CMSActionBaseAPIModel
	var obj CMSActionAPIBaseAccessor
	rawBody := mp.peekBody(req, &base)
	switch base.Type {
	case "deploy":
		da := &CMSActionDeployAPIModel{}
		err := json.Unmarshal(rawBody, &da)
		assert.NoError(mp.t, err)
		da.Output = &CMSDeployActionOutputAPIModel{
			TransactionID:  nanoid.New(),
			IdempotencyKey: nanoid.New(),
			OperationID:    nanoid.New(),
			Location: CMSDeployActionOutputLocationAPIModel{
				Address: nanoid.New(),
			},
			BlockNumber: "12345",
		}
		obj = da
	case "createapi":
		da := &CMSActionCreateAPIAPIModel{}
		err := json.Unmarshal(rawBody, &da)
		assert.NoError(mp.t, err)
		da.Output = &CMSCreateAPIActionOutputAPIModel{
			APIID: nanoid.New(),
		}
		obj = da
	}
	base = obj.ActionBase()
	base.ID = nanoid.New()
	now := time.Now().UTC()
	base.Created = &now
	base.Updated = &now
	mp.cmsActions[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+base.ID] = obj
	mp.respond(res, &obj, 201)
}

func (mp *mockPlatform) patchCMSAction(res http.ResponseWriter, req *http.Request) {
	obj := mp.cmsActions[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["action"]] // expected behavior of provider is PUT only on exists
	assert.NotNil(mp.t, obj)
	var updates CMSActionDeployAPIModel
	mp.getBody(req, &updates)
	assert.Empty(mp.t, updates.ID)
	now := time.Now().UTC()
	obj.ActionBase().Created = &now
	obj.ActionBase().Updated = &now
	obj.OutputBase().Status = "pending"
	mp.respond(res, &obj, 200)
}

func (mp *mockPlatform) deleteCMSAction(res http.ResponseWriter, req *http.Request) {
	obj := mp.cmsActions[mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["action"]]
	assert.NotNil(mp.t, obj)
	delete(mp.cmsActions, mux.Vars(req)["env"]+"/"+mux.Vars(req)["service"]+"/"+mux.Vars(req)["action"])
	mp.respond(res, nil, 204)
}
