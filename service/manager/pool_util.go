//    Copyright 2023 Ewout Prangsma
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package manager

import (
	"time"

	"github.com/mattn/go-pubsub"
	"github.com/rs/zerolog"
)

const (
	pubTimeout = time.Millisecond * 20
)

// safePub publishes the given value into the given pool, checking how long it took to publish.
func safePub(log zerolog.Logger, pool *pubsub.PubSub, value interface{}) {
	start := time.Now()
	pool.Pub(value)
	if delta := time.Since(start); delta > pubTimeout {
		log.Error().
			Str("timeout", pubTimeout.String()).
			Str("actual", delta.String()).
			Msg("Pub into pool took too long")
	}
}
