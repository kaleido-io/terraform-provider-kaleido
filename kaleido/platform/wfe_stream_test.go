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
	"net/http"
	"testing"
	"time"

	"github.com/aidarkhanov/nanoid"
	"github.com/gorilla/mux"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const wfe_stream_step1 = `
resource "kaleido_platform_wfe_stream" "test_stream" {
  environment = "test-env"
  service = "test-service"
  name = "test-stream"
  uniqueness_prefix = "evmtx."
  listener_handler = "evmTxnFinalizations"
  listener_handler_provider = "s-12345abcde"
  description = "Test stream for workflow engine"
  type = "correlation_stream"
  config = jsonencode({
      "fromBlock" = "latest"
      "batchSize" = 50
      "batchTimeout" = "500ms"
      "pollTimeout" = "2s"
  })
}
`

const wfe_stream_step2 = `
resource "kaleido_platform_wfe_stream" "test_stream" {
  environment = "test-env"
  service = "test-service"
  name = "test-stream"
  uniqueness_prefix = "evmtx."
  listener_handler = "evmTxnFinalizations"
  listener_handler_provider = "s-12345abcde"
  description = "Test stream for workflow engine - updated"
  type = "correlation_stream"
  config = jsonencode({
      "fromBlock" = "latest"
      "batchSize" = 150
      "batchTimeout" = "500ms"
      "pollTimeout" = "2s"
  })
}
`

const wfe_stream_resource = "kaleido_platform_wfe_stream.test_stream"

func TestWFEStream1(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"PUT /endpoint/{env}/{service}/rest/api/v1/streams/{stream}",
			"GET /endpoint/{env}/{service}/rest/api/v1/streams/{stream}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/streams/{stream}",
			"GET /endpoint/{env}/{service}/rest/api/v1/streams/{stream}",
		})
		mp.server.Close()
	}()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + wfe_stream_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(wfe_stream_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(wfe_stream_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(wfe_stream_resource, "name", "test-stream"),
					resource.TestCheckResourceAttr(wfe_stream_resource, "uniqueness_prefix", "evmtx."),
					resource.TestCheckResourceAttr(wfe_stream_resource, "listener_handler", "evmTxnFinalizations"),
					resource.TestCheckResourceAttr(wfe_stream_resource, "listener_handler_provider", "s-12345abcde"),
					resource.TestCheckResourceAttr(wfe_stream_resource, "type", "correlation_stream"),
					resource.TestCheckResourceAttr(wfe_stream_resource, "description", "Test stream for workflow engine"),
					resource.TestCheckResourceAttrSet(wfe_stream_resource, "id"),
					resource.TestCheckResourceAttrSet(wfe_stream_resource, "created"),
					resource.TestCheckResourceAttrSet(wfe_stream_resource, "updated"),
				),
			},
		},
	})
}

func TestWFEStream2(t *testing.T) {
	mp, providerConfig := testSetup(t)
	defer func() {
		mp.checkClearCalls([]string{
			"PUT /endpoint/{env}/{service}/rest/api/v1/streams/{stream}",
			"GET /endpoint/{env}/{service}/rest/api/v1/streams/{stream}",
			"GET /endpoint/{env}/{service}/rest/api/v1/streams/{stream}",
			"PATCH /endpoint/{env}/{service}/rest/api/v1/streams/{stream}",
			"GET /endpoint/{env}/{service}/rest/api/v1/streams/{stream}",
			"DELETE /endpoint/{env}/{service}/rest/api/v1/streams/{stream}",
			"GET /endpoint/{env}/{service}/rest/api/v1/streams/{stream}",
		})
		mp.server.Close()
	}()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + wfe_stream_step1,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(wfe_stream_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(wfe_stream_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(wfe_stream_resource, "name", "test-stream"),
					resource.TestCheckResourceAttr(wfe_stream_resource, "uniqueness_prefix", "evmtx."),
					resource.TestCheckResourceAttr(wfe_stream_resource, "listener_handler", "evmTxnFinalizations"),
					resource.TestCheckResourceAttr(wfe_stream_resource, "listener_handler_provider", "s-12345abcde"),
					resource.TestCheckResourceAttr(wfe_stream_resource, "type", "correlation_stream"),
					resource.TestCheckResourceAttr(wfe_stream_resource, "description", "Test stream for workflow engine"),
					resource.TestCheckResourceAttrSet(wfe_stream_resource, "id"),
					resource.TestCheckResourceAttrSet(wfe_stream_resource, "created"),
					resource.TestCheckResourceAttrSet(wfe_stream_resource, "updated"),
				),
			},
			{
				Config: providerConfig + wfe_stream_step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(wfe_stream_resource, "environment", "test-env"),
					resource.TestCheckResourceAttr(wfe_stream_resource, "service", "test-service"),
					resource.TestCheckResourceAttr(wfe_stream_resource, "name", "test-stream"),
					resource.TestCheckResourceAttr(wfe_stream_resource, "description", "Test stream for workflow engine - updated"),
					resource.TestCheckResourceAttrSet(wfe_stream_resource, "id"),
					resource.TestCheckResourceAttrSet(wfe_stream_resource, "created"),
					resource.TestCheckResourceAttrSet(wfe_stream_resource, "updated"),
				),
			},
		},
	})
}

// Mock handlers for streams
func (mp *mockPlatform) putWFEStream(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	streamNameOrID := vars["stream"]
	stream, exists := mp.wfeStreams[streamNameOrID]
	newStream := new(WFEStreamAPIModel)
	mp.getBody(req, newStream)
	now := time.Now().UTC()
	if !exists {
		newStream.ID = nanoid.New()
		newStream.Created = &now
	} else {
		newStream.ID = stream.ID
		newStream.Name = stream.Name
		newStream.UniquenessPrefix = stream.UniquenessPrefix
		newStream.ListenerHandler = stream.ListenerHandler
		newStream.ListenerHandlerProvider = stream.ListenerHandlerProvider
		newStream.Type = stream.Type
	}
	newStream.Updated = &now
	// Store by both ID and name for lookup
	mp.wfeStreams[newStream.Name] = newStream
	mp.wfeStreams[newStream.ID] = newStream
	mp.respond(res, newStream, http.StatusOK)
}

func (mp *mockPlatform) getWFEStream(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	streamNameOrID := vars["stream"]
	stream, exists := mp.wfeStreams[streamNameOrID]
	if !exists {
		// Try to find by name
		for _, s := range mp.wfeStreams {
			if s.Name == streamNameOrID {
				stream = s
				exists = true
				break
			}
		}
	}
	if !exists {
		mp.respond(res, nil, 404)
		return
	}

	mp.respond(res, stream, http.StatusOK)
}

func (mp *mockPlatform) patchWFEStream(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	streamNameOrID := vars["stream"]
	stream, exists := mp.wfeStreams[streamNameOrID]
	if !exists {
		mp.respond(res, nil, 404)
		return
	}
	var streamUpdates WFEStreamAPIModel
	mp.getBody(req, &streamUpdates)
	if streamUpdates.Name != "" {
		stream.Name = streamUpdates.Name
	}
	if streamUpdates.Description != "" {
		stream.Description = streamUpdates.Description
	}
	if streamUpdates.UniquenessPrefix != "" {
		stream.UniquenessPrefix = streamUpdates.UniquenessPrefix
	}
	if streamUpdates.ListenerHandler != "" {
		stream.ListenerHandler = streamUpdates.ListenerHandler
	}
	if streamUpdates.ListenerHandlerProvider != "" {
		stream.ListenerHandlerProvider = streamUpdates.ListenerHandlerProvider
	}
	if streamUpdates.Type != "" {
		stream.Type = streamUpdates.Type
	}
	if streamUpdates.Config != nil {
		stream.Config = streamUpdates.Config
	}
	now := time.Now().UTC()
	stream.Updated = &now
	// Store by both ID and name for lookup
	mp.wfeStreams[stream.Name] = stream
	mp.wfeStreams[stream.ID] = stream
	mp.respond(res, stream, http.StatusOK)
}

func (mp *mockPlatform) deleteWFEStream(res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	streamID := vars["stream"]
	stream, exists := mp.wfeStreams[streamID]
	if !exists {
		mp.respond(res, nil, 404)
		return
	}
	delete(mp.wfeStreams, streamID)
	delete(mp.wfeStreams, stream.Name)
	mp.respond(res, nil, http.StatusNoContent)
}
