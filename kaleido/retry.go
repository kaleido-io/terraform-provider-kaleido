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
package kaleido

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Note: https://discuss.hashicorp.com/t/terraform-plugin-framework-what-is-the-replacement-for-waitforstate-or-retrycontext/45538
type retry struct {
	InitialDelay time.Duration
	MaximumDelay time.Duration
	Factor       float64
}

var Retry = &retry{
	InitialDelay: 500 * time.Millisecond,
	MaximumDelay: 5 * time.Second,
	Factor:       2.0,
}

// Simple retry handler
func (r *retry) Do(ctx context.Context, logDescription string, f func(attempt int) (retry bool, err error)) error {
	attempt := 0
	delay := r.InitialDelay
	factor := r.Factor
	for {
		attempt++
		retry, err := f(attempt)
		if err != nil {
			tflog.Error(ctx, fmt.Sprintf("%s attempt %d: %s", logDescription, attempt, err))
		}
		if !retry || err == nil {
			return err
		}

		// Check the context isn't canceled
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled")
		default:
		}

		// Limit the delay based on the context deadline and maximum delay
		deadline, dok := ctx.Deadline()
		now := time.Now()
		if delay > r.MaximumDelay {
			delay = r.MaximumDelay
		}
		if dok {
			timeleft := deadline.Sub(now)
			if timeleft < delay {
				delay = timeleft
			}
		}

		// Sleep and set the delay for next time
		time.Sleep(delay)
		delay = time.Duration(float64(delay) * factor)
	}
}
