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

type powerPool struct {
	mutex          sync.RWMutex
	power          api.Power
	requestChanges *pubsub.PubSub
	actualChanges  *pubsub.PubSub
}

func newPowerPool() *powerPool {
	return &powerPool{
		power: api.Power{
			Request: &api.PowerState{},
			Actual:  &api.PowerState{},
		},
		requestChanges: pubsub.New(),
		actualChanges:  pubsub.New(),
	}
}

func (p *powerPool) SetRequest(x api.PowerState) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.power.Request.Enabled = x.GetEnabled()
	p.requestChanges.Pub(p.power.Clone())
}

func (p *powerPool) SetActual(x api.PowerState) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.power.Actual.Enabled = x.GetEnabled()
	p.actualChanges.Pub(p.power.Clone())
}

func (p *powerPool) SubRequest(enabled bool) (chan api.Power, context.CancelFunc) {
	c := make(chan api.Power)
	if enabled {
		cb := func(msg *api.Power) {
			c <- *msg
		}
		p.requestChanges.Sub(cb)
		return c, func() {
			p.requestChanges.Leave(cb)
			close(c)
		}
	} else {
		return c, func() {
			close(c)
		}
	}
}

func (p *powerPool) SubActual(enabled bool) (chan api.Power, context.CancelFunc) {
	c := make(chan api.Power)
	if enabled {
		cb := func(msg *api.Power) {
			c <- *msg
		}
		p.actualChanges.Sub(cb)
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
