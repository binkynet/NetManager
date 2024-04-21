//    Copyright 2020 Ewout Prangsma
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
	"context"
	"sync"
	"time"

	api "github.com/binkynet/BinkyNet/apis/v1"
	"github.com/mattn/go-pubsub"
	"github.com/rs/zerolog"
)

type powerPool struct {
	mutex          sync.RWMutex
	log            zerolog.Logger
	power          api.Power
	requestChanges *pubsub.PubSub
	actualChanges  *pubsub.PubSub
}

func newPowerPool(log zerolog.Logger) *powerPool {
	return &powerPool{
		log: log.With().Str("pool", "power").Logger(),
		power: api.Power{
			Request: &api.PowerState{},
			Actual:  &api.PowerState{},
		},
		requestChanges: pubsub.New(),
		actualChanges:  pubsub.New(),
	}
}

func (p *powerPool) SetRequest(x api.PowerState) {
	powerPoolMetrics.SetRequestTotalCounters.WithLabelValues("power").Inc()
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.power.Request.Enabled = x.GetEnabled()
	safePub(p.log, p.requestChanges, p.power.Clone())
}

func (p *powerPool) SetActual(x api.PowerState) {
	powerPoolMetrics.SetActualTotalCounters.WithLabelValues("power").Inc()
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.power.Actual.Enabled = x.GetEnabled()
	safePub(p.log, p.actualChanges, p.power.Clone())
}

func (p *powerPool) SubActual(enabled bool, timeout time.Duration) (chan api.Power, context.CancelFunc) {
	powerPoolMetrics.SubActualTotalCounter.Inc()
	c := make(chan api.Power)
	if enabled {
		// Subscribe
		cb := func(msg *api.Power) {
			select {
			case c <- *msg:
				// Done
				powerPoolMetrics.SubActualMessagesTotalCounters.WithLabelValues("power").Inc()
			case <-time.After(timeout):
				p.log.Error().
					Dur("timeout", timeout).
					Msg("Failed to deliver power actual to channel")
				powerPoolMetrics.SubActualMessagesFailedTotalCounters.WithLabelValues("power").Inc()
			}
		}
		p.actualChanges.Sub(cb)
		// Publish known request state
		p.mutex.RLock()
		go cb(p.power.Clone())
		p.mutex.RUnlock()
		// Return channel & cancel function
		return c, func() {
			p.actualChanges.Leave(cb)
			close(c)
		}
	} else {
		return c, func() {
			close(c)
		}
	}
}
