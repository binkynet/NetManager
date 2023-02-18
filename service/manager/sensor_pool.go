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

type sensorPool struct {
	mutex         sync.RWMutex
	log           zerolog.Logger
	entries       map[api.ObjectAddress]*api.Sensor
	actualChanges *pubsub.PubSub
}

func newSensorPool(log zerolog.Logger) *sensorPool {
	return &sensorPool{
		entries:       make(map[api.ObjectAddress]*api.Sensor),
		log:           log.With().Str("pool", "sensor").Logger(),
		actualChanges: pubsub.New(),
	}
}

func (p *sensorPool) SetActual(x api.Sensor) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	e, found := p.entries[x.Address]
	if !found {
		e = x.Clone()
		p.entries[x.Address] = e
	} else {
		e.Actual = x.GetActual().Clone()
	}
	p.actualChanges.Pub(e.Clone())
}

func (p *sensorPool) SubActual(enabled bool, timeout time.Duration, filter ModuleFilter) (chan api.Sensor, context.CancelFunc) {
	c := make(chan api.Sensor)
	if enabled {
		// Subscribe
		cb := func(msg *api.Sensor) {
			if filter.Matches(msg.GetAddress()) {
				select {
				case c <- *msg:
					// Done
				case <-time.After(timeout):
					p.log.Error().
						Dur("timeout", timeout).
						Str("address", string(msg.GetAddress())).
						Msg("Failed to deliver sensor actual to channel")
				}
			}
		}
		p.actualChanges.Sub(cb)
		// Publish all known actual states
		p.mutex.RLock()
		for _, sensor := range p.entries {
			if sensor.GetActual() != nil && filter.Matches(sensor.GetAddress()) {
				cb(sensor.Clone())
			}
		}
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
