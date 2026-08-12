// Copyright © Kaleido, Inc. 2018, 2026

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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildUserAgent(t *testing.T) {
	// Release build: goreleaser injects version and full commit hash
	assert.Equal(t, "Terraform / 1.3.2+abcdef01 (Platform)",
		buildUserAgent("1.3.2", "abcdef0123456789abcdef0123456789abcdef01", "Platform"))

	// Short commit is used as-is
	assert.Equal(t, "Terraform / 1.3.2+abc123 (BaaS)",
		buildUserAgent("1.3.2", "abc123", "BaaS"))

	// Local build: no ldflags at all
	assert.Equal(t, "Terraform / dev (Platform)",
		buildUserAgent("", "", "Platform"))

	// Version set but no commit
	assert.Equal(t, "Terraform / dev (Platform)",
		buildUserAgent("dev", "", "Platform"))
}
