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

package service

import (
	"context"
	"sync"

	api "github.com/binkynet/BinkyNet/apis/v1"
	"github.com/mattn/go-pubsub"
)

type clockPool struct {
	mutex         sync.RWMutex
	clock         api.Clock
	actualChanges *pubsub.PubSub
}

func newClockPool() *clockPool {
	return &clockPool{
		actualChanges: pubsub.New(),
	}
}

func (p *clockPool) SetActual(x api.Clock) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.clock.Period = x.GetPeriod()
	p.clock.Hours = x.GetHours()
	p.clock.Minutes = x.GetMinutes()
	p.actualChanges.Pub(p.clock.Clone())
}

func (p *clockPool) SubActual() (chan api.Clock, context.CancelFunc) {
	c := make(chan api.Clock)
	cb := func(msg *api.Clock) {
		c <- *msg
	}
	p.actualChanges.Sub(cb)
	return c, func() {
		p.actualChanges.Leave(cb)
		close(c)
	}
}
