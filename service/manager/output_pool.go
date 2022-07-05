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

type outputPool struct {
	mutex          sync.RWMutex
	entries        map[api.ObjectAddress]*api.Output
	requestChanges *pubsub.PubSub
	actualChanges  *pubsub.PubSub
}

func newOutputPool() *outputPool {
	return &outputPool{
		entries:        make(map[api.ObjectAddress]*api.Output),
		requestChanges: pubsub.New(),
		actualChanges:  pubsub.New(),
	}
}

func (p *outputPool) SetRequest(x api.Output) {
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

func (p *outputPool) SetActual(x api.Output) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	e, found := p.entries[x.Address]
	if !found {
		// Check global address
		_, local, _ := api.SplitAddress(x.Address)
		globalAddr := api.JoinModuleLocal(api.GlobalModuleID, local)
		e, found = p.entries[globalAddr]
		if !found {
			// Apparently we do not care for this output
			return
		}
	}
	e.Actual = x.GetActual().Clone()
	p.actualChanges.Pub(e.Clone())
}

func (p *outputPool) SubRequest(enabled bool) (chan api.Output, context.CancelFunc) {
	c := make(chan api.Output)
	if enabled {
		// Subscribe
		cb := func(msg *api.Output) {
			c <- *msg
		}
		p.requestChanges.Sub(cb)
		// Publish all known request states
		p.mutex.RLock()
		for _, output := range p.entries {
			if output.GetRequest() != nil {
				p.requestChanges.Sub(output.Clone())
			}
		}
		p.mutex.RUnlock()
		// Return channel & cancel function
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

func (p *outputPool) SubActual(enabled bool) (chan api.Output, context.CancelFunc) {
	c := make(chan api.Output)
	if enabled {
		// Subscribe
		cb := func(msg *api.Output) {
			c <- *msg
		}
		p.actualChanges.Sub(cb)
		// Publish all known actual states
		p.mutex.RLock()
		for _, output := range p.entries {
			if output.GetActual() != nil {
				p.actualChanges.Sub(output.Clone())
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
