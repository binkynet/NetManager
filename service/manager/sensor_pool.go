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

	api "github.com/binkynet/BinkyNet/apis/v1"
	"github.com/mattn/go-pubsub"
)

type sensorPool struct {
	mutex         sync.RWMutex
	entries       map[api.ObjectAddress]*api.Sensor
	actualChanges *pubsub.PubSub
}

func newSensorPool() *sensorPool {
	return &sensorPool{
		entries:       make(map[api.ObjectAddress]*api.Sensor),
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

func (p *sensorPool) SubActual(enabled bool, filter ModuleFilter) (chan api.Sensor, context.CancelFunc) {
	c := make(chan api.Sensor)
	if enabled {
		// Subscribe
		cb := func(msg *api.Sensor) {
			if filter.Matches(msg.GetAddress()) {
				c <- *msg
			}
		}
		p.actualChanges.Sub(cb)
		// Publish all known actual states
		p.mutex.RLock()
		for _, sensor := range p.entries {
			if sensor.GetActual() != nil && filter.Matches(sensor.GetAddress()) {
				p.actualChanges.Sub(sensor.Clone())
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
