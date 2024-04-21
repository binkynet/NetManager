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

type locPool struct {
	mutex         sync.RWMutex
	log           zerolog.Logger
	entries       map[api.ObjectAddress]*api.Loc
	actualChanges *pubsub.PubSub
}

func newLocPool(log zerolog.Logger) *locPool {
	return &locPool{
		entries:       make(map[api.ObjectAddress]*api.Loc),
		log:           log.With().Str("pool", "loc").Logger(),
		actualChanges: pubsub.New(),
	}
}

func (p *locPool) SetRequest(x api.Loc) {
	locPoolMetrics.SetRequestTotalCounters.WithLabelValues(string(x.GetAddress())).Inc()
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
}

func (p *locPool) SetActual(x api.Loc) {
	locPoolMetrics.SetActualTotalCounters.WithLabelValues(string(x.GetAddress())).Inc()
	p.mutex.Lock()
	defer p.mutex.Unlock()

	e, found := p.entries[x.Address]
	if !found {
		// Apparently we do not care for this loc
		return
	} else {
		e.Actual = x.GetActual().Clone()
	}
	safePub(p.log, p.actualChanges, e.Clone())
}

func (p *locPool) SubActual(enabled bool, timeout time.Duration) (chan api.Loc, context.CancelFunc) {
	locPoolMetrics.SubActualTotalCounter.Inc()
	c := make(chan api.Loc)
	if enabled {
		// Subscribe
		cb := func(msg *api.Loc) {
			select {
			case c <- *msg:
				// Done
				locPoolMetrics.SubActualMessagesTotalCounters.WithLabelValues(string(msg.GetAddress())).Inc()
			case <-time.After(timeout):
				p.log.Error().
					Dur("timeout", timeout).
					Str("address", string(msg.GetAddress())).
					Msg("Failed to deliver loc actual to channel")
				locPoolMetrics.SubActualMessagesFailedTotalCounters.WithLabelValues(string(msg.GetAddress())).Inc()
			}
		}
		p.actualChanges.Sub(cb)
		// Publish all known actual states
		p.mutex.RLock()
		for _, loc := range p.entries {
			if loc.GetActual() != nil {
				go cb(loc.Clone())
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
