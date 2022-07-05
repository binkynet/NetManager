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

type locPool struct {
	mutex          sync.RWMutex
	entries        map[api.ObjectAddress]*api.Loc
	requestChanges *pubsub.PubSub
	actualChanges  *pubsub.PubSub
}

func newLocPool() *locPool {
	return &locPool{
		entries:        make(map[api.ObjectAddress]*api.Loc),
		requestChanges: pubsub.New(),
		actualChanges:  pubsub.New(),
	}
}

func (p *locPool) SetRequest(x api.Loc) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	e, found := p.entries[x.Address]
	if !found {
		x.Actual = nil
		e = x.Clone()
		p.entries[x.Address] = e
	} else {
		e.Request = x.GetRequest().Clone()
	}
	p.requestChanges.Pub(e.Clone())
}

func (p *locPool) SetActual(x api.Loc) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	e, found := p.entries[x.Address]
	if !found {
		// Apparently we do not care for this loc
		return
	} else {
		e.Actual = x.GetActual().Clone()
	}
	p.actualChanges.Pub(e.Clone())
}

func (p *locPool) SubRequest(enabled bool) (chan api.Loc, context.CancelFunc) {
	c := make(chan api.Loc)
	if enabled {
		cb := func(msg *api.Loc) {
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

func (p *locPool) SubActual(enabled bool) (chan api.Loc, context.CancelFunc) {
	c := make(chan api.Loc)
	if enabled {
		cb := func(msg *api.Loc) {
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
