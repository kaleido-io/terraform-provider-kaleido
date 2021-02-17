// Copyright 2020 Kaleido, a ConsenSys business

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package kaleido

import (
	"encoding/json"
	"net/http"
	"strings"

	kaleido "github.com/kaleido-io/kaleido-sdk-go/kaleido"
	gock "gopkg.in/h2non/gock.v1"
)

func testDebugGocks() {
	gock.Observe(gock.DumpRequest)
}

func testAvoidPartialRegexMatch(mustEndWith string) gock.MatchFunc {
	return func(req *http.Request, gReq *gock.Request) (bool, error) {
		// Avoid a partial regex match on sub-resources
		return strings.HasSuffix(req.URL.Path, mustEndWith), nil
	}
}

func jsonClone(input interface{}) map[string]interface{} {
	var output map[string]interface{}
	b, _ := json.Marshal(input)
	json.Unmarshal(b, &output)
	return output
}

func testConsortiumGocks(consortium *kaleido.Consortium) {

	consortiumCreateRequest := jsonClone(consortium)
	consortiumCreateResponse := jsonClone(consortiumCreateRequest)
	consortiumCreateResponse["_id"] = "cons1"
	consortiumGetResponse1 := jsonClone(consortiumCreateResponse)

	gock.New("http://example.com").
		AddMatcher(testAvoidPartialRegexMatch("/consortia")).
		Post("/api/v1/consortia").
		MatchType("json").
		JSON(consortiumCreateRequest).
		Reply(201).
		JSON(consortiumCreateResponse)

	gock.New("http://example.com").
		AddMatcher(testAvoidPartialRegexMatch("/consortia/cons1")).
		Get("/api/v1/consortia/cons1").
		Persist().
		Reply(200).
		JSON(consortiumGetResponse1)

	gock.New("http://example.com").
		AddMatcher(testAvoidPartialRegexMatch("/consortia/cons1")).
		Delete("/api/v1/consortia/cons1").
		Reply(202)

}

func testMembershipGocks(membership *kaleido.Membership) {

	membershipCreateRequest := jsonClone(membership)
	membershipCreateResponse := jsonClone(membershipCreateRequest)
	membershipCreateResponse["_id"] = "member1"
	membershipGetResponse1 := jsonClone(membershipCreateResponse)

	gock.New("http://example.com").
		Post("/api/v1/consortia/cons1/memberships").
		MatchType("json").
		JSON(membershipCreateRequest).
		Reply(201).
		JSON(membershipCreateResponse)

	gock.New("http://example.com").
		Get("/api/v1/consortia/cons1/memberships/member1").
		Persist().
		Reply(200).
		JSON(membershipGetResponse1)

	gock.New("http://example.com").
		Delete("/api/v1/consortia/cons1/memberships/member1").
		Reply(409). // Force a retry
		JSON(membershipGetResponse1)

	gock.New("http://example.com").
		Delete("/api/v1/consortia/cons1/memberships/member1").
		Reply(204)

}

func testEnvironmentGocks(environment *kaleido.Environment) {

	environmentCreateRequest := jsonClone(environment)
	environmentCreateResponse := jsonClone(environmentCreateRequest)
	environmentCreateResponse["_id"] = "env1"
	environmentCreateResponse["state"] = "initializing"
	environmentGetResponse1 := jsonClone(environmentCreateResponse)
	environmentGetResponse2 := jsonClone(environmentCreateResponse)
	environmentGetResponse2["state"] = "live"

	gock.New("http://example.com").
		AddMatcher(testAvoidPartialRegexMatch("/environments")).
		Post("/api/v1/consortia/cons1/environments").
		MatchType("json").
		JSON(environmentCreateRequest).
		Reply(201).
		JSON(environmentCreateResponse)

	gock.New("http://example.com").
		AddMatcher(testAvoidPartialRegexMatch("/environments/env1")).
		Get("/api/v1/consortia/cons1/environments/env1").
		Reply(200).
		JSON(environmentGetResponse1)

	gock.New("http://example.com").
		AddMatcher(testAvoidPartialRegexMatch("/environments/env1")).
		Get("/api/v1/consortia/cons1/environments/env1").
		Persist().
		Reply(200).
		JSON(environmentGetResponse2)

	gock.New("http://example.com").
		AddMatcher(testAvoidPartialRegexMatch("/environments/env1")).
		Delete("/api/v1/consortia/cons1/environments/env1").
		Reply(204)

}

func testEZoneGocks(ezone *kaleido.EZone) {

	ezoneCreateRequest := jsonClone(ezone)
	ezoneCreateResponse := jsonClone(ezoneCreateRequest)
	ezoneCreateResponse["_id"] = "zone1"
	ezoneGetResponse1 := jsonClone(ezoneCreateResponse)

	gock.New("http://example.com").
		Post("/api/v1/consortia/cons1/environments/env1/zones").
		MatchType("json").
		JSON(ezoneCreateRequest).
		Reply(201).
		JSON(ezoneCreateResponse)

	gock.New("http://example.com").
		Get("/api/v1/consortia/cons1/environments/env1/zones/zone1").
		Persist().
		Reply(200).
		JSON(ezoneGetResponse1)

	gock.New("http://example.com").
		Delete("/api/v1/consortia/cons1/environments/env1/zones/zone1").
		Reply(204)

}

func testServiceGocks(service *kaleido.Service) {

	serviceCreateRequest := jsonClone(service)
	serviceCreateResponse := jsonClone(serviceCreateRequest)
	serviceCreateResponse["_id"] = "svc1"
	serviceCreateResponse["state"] = "provisioning"
	serviceGetResponse1 := jsonClone(serviceCreateResponse)
	serviceGetResponse2 := jsonClone(serviceCreateResponse)
	serviceGetResponse2["state"] = "started"
	serviceGetResponse2["urls"] = map[string]string{
		"http": "http://test-http.example.com",
	}

	gock.New("http://example.com").
		Post("/api/v1/consortia/cons1/environments/env1/services").
		MatchType("json").
		JSON(serviceCreateRequest).
		Reply(201).
		JSON(serviceCreateResponse)

	gock.New("http://example.com").
		Get("/api/v1/consortia/cons1/environments/env1/services/svc1").
		Reply(200).
		JSON(serviceGetResponse1)

	gock.New("http://example.com").
		Get("/api/v1/consortia/cons1/environments/env1/services/svc1").
		Persist().
		Reply(200).
		JSON(serviceGetResponse2)

	gock.New("http://example.com").
		Delete("/api/v1/consortia/cons1/environments/env1/services/svc1").
		Reply(204)

}

func testIPFSServiceGocks(service *kaleido.Service) {

	serviceCreateRequest := jsonClone(service)
	serviceCreateResponse := jsonClone(serviceCreateRequest)
	serviceCreateResponse["_id"] = "ipfs_svc1"
	serviceCreateResponse["state"] = "provisioning"
	serviceGetResponse1 := jsonClone(serviceCreateResponse)
	serviceGetResponse2 := jsonClone(serviceCreateResponse)
	serviceGetResponse2["state"] = "started"
	serviceGetResponse2["urls"] = map[string]string{
		"http":          "http://test-http.example.com",
		"webui":         "http://test-http.example.com",
		"websocket_url": "http://test-http.example.com",
	}

	gock.New("http://example.com").
		Post("/api/v1/consortia/cons1/environments/env1/services").
		MatchType("json").
		JSON(serviceCreateRequest).
		Reply(201).
		JSON(serviceCreateResponse)

	gock.New("http://example.com").
		Get("/api/v1/consortia/cons1/environments/env1/services/ipfs_svc1").
		Reply(200).
		JSON(serviceGetResponse1)

	gock.New("http://example.com").
		Get("/api/v1/consortia/cons1/environments/env1/services/ipfs_svc1").
		Persist().
		Reply(200).
		JSON(serviceGetResponse2)

	gock.New("http://example.com").
		Delete("/api/v1/consortia/cons1/environments/env1/services/ipfs_svc1").
		Reply(204)

}

func testNodeGocks(node *kaleido.Node) {

	nodeCreateRequest := jsonClone(node)
	nodeCreateResponse := jsonClone(nodeCreateRequest)
	nodeCreateResponse["_id"] = "node1"
	nodeCreateResponse["state"] = "initializing"
	nodeGetResponse1 := jsonClone(nodeCreateResponse)
	nodeGetResponse2 := jsonClone(nodeCreateResponse)
	nodeGetResponse2["state"] = "started"
	nodeGetResponse2["urls"] = map[string]string{
		"rpc": "http://test-rpc.example.com",
		"wss": "http://test-wss.example.com",
	}

	gock.New("http://example.com").
		Post("/api/v1/consortia/cons1/environments/env1/nodes").
		MatchType("json").
		JSON(nodeCreateRequest).
		Reply(201).
		JSON(nodeCreateResponse)

	gock.New("http://example.com").
		Get("/api/v1/consortia/cons1/environments/env1/nodes/node1").
		Reply(200).
		JSON(nodeGetResponse1)

	gock.New("http://example.com").
		Get("/api/v1/consortia/cons1/environments/env1/nodes/node1").
		Persist().
		Reply(200).
		JSON(nodeGetResponse2)

	gock.New("http://example.com").
		Delete("/api/v1/consortia/cons1/environments/env1/nodes/node1").
		Reply(204)

}

func testConfigurationGocks(configuration *kaleido.Configuration) {

	configurationCreateRequest := jsonClone(configuration)
	configurationCreateResponse := jsonClone(configurationCreateRequest)
	configurationCreateResponse["_id"] = "cfg1"
	configurationGetResponse1 := jsonClone(configurationCreateResponse)

	gock.New("http://example.com").
		Post("/api/v1/consortia/cons1/environments/env1/configurations").
		MatchType("json").
		JSON(configurationCreateRequest).
		Reply(201).
		JSON(configurationCreateResponse)

	gock.New("http://example.com").
		Get("/api/v1/consortia/cons1/environments/env1/configurations/cfg1").
		Persist().
		Reply(200).
		JSON(configurationGetResponse1)

	gock.New("http://example.com").
		Delete("/api/v1/consortia/cons1/environments/env1/configurations/cfg1").
		Reply(204)

}
