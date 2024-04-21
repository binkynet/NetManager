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

type switchPool struct {
	mutex   sync.RWMutex
	log     zerolog.Logger
	entries map[api.ObjectAddress]*api.Switch
	//requestChanges *pubsub.PubSub
	actualChanges *pubsub.PubSub
}

func newSwitchPool(log zerolog.Logger) *switchPool {
	return &switchPool{
		entries: make(map[api.ObjectAddress]*api.Switch),
		log:     log.With().Str("pool", "switch").Logger(),
		//requestChanges: pubsub.New(),
		actualChanges: pubsub.New(),
	}
}

func (p *switchPool) SetRequest(x api.Switch) {
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
	switchPoolMetrics.SetRequestTotalCounters.WithLabelValues(string(x.Address)).Inc()
}

func (p *switchPool) SetActual(x api.Switch) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	e, found := p.entries[x.Address]
	if !found {
		// Check global address
		_, local, _ := api.SplitAddress(x.Address)
		globalAddr := api.JoinModuleLocal(api.GlobalModuleID, local)
		e, found = p.entries[globalAddr]
		if !found {
			// Apparently we do not care for this switch
			return
		}
	}
	e.Actual = x.GetActual().Clone()
	safePub(p.log, p.actualChanges, e.Clone())
	switchPoolMetrics.SetActualTotalCounters.WithLabelValues(string(x.Address)).Inc()
}

func (p *switchPool) SubActual(enabled bool, timeout time.Duration, filter ModuleFilter) (chan api.Switch, context.CancelFunc) {
	switchPoolMetrics.SubActualTotalCounter.Inc()
	c := make(chan api.Switch)
	if enabled {
		// Subscribe
		cb := func(msg *api.Switch) {
			if filter.Matches(msg.GetAddress()) {
				select {
				case c <- *msg:
					// Done
					switchPoolMetrics.SubActualMessagesTotalCounters.WithLabelValues(string(msg.GetAddress())).Inc()
				case <-time.After(timeout):
					p.log.Error().
						Dur("timeout", timeout).
						Str("address", string(msg.GetAddress())).
						Msg("Failed to deliver switch actual to channel")
					switchPoolMetrics.SubActualMessagesFailedTotalCounters.WithLabelValues(string(msg.GetAddress())).Inc()
				}
			}
		}
		p.actualChanges.Sub(cb)
		// Publish all known actual states
		p.mutex.RLock()
		for _, sw := range p.entries {
			if sw.GetActual() != nil && filter.Matches(sw.GetAddress()) {
				go cb(sw.Clone())
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
