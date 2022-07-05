//    Copyright 2021 Ewout Prangsma
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
	"sort"
	"sync"
	"time"

	api "github.com/binkynet/BinkyNet/apis/v1"
	"github.com/mattn/go-pubsub"
	"github.com/rs/zerolog"
)

type localWorkerPool struct {
	log      zerolog.Logger
	mutex    sync.RWMutex
	requests *pubsub.PubSub
	actuals  *pubsub.PubSub
	workers  map[string]*localWorkerEntry
}

type localWorkerEntry struct {
	api.LocalWorker
	lastUpdatedActualAt time.Time
}

func newLocalWorkerPool(log zerolog.Logger) *localWorkerPool {
	return &localWorkerPool{
		log:      log,
		requests: pubsub.New(),
		actuals:  pubsub.New(),
		workers:  make(map[string]*localWorkerEntry),
	}
}

// GetInfo fetches the last known info for a local worker with given ID.
// Returns: info, lastUpdatedAt, found
func (p *localWorkerPool) GetInfo(id string) (api.LocalWorkerInfo, time.Time, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if lw, found := p.workers[id]; found {
		if info := lw.GetActual(); info != nil {
			return *info, lw.lastUpdatedActualAt, found
		}
	}
	return api.LocalWorkerInfo{}, time.Time{}, false
}

// GetAll fetches the last known info for all local workers.
func (p *localWorkerPool) GetAll() []api.LocalWorkerInfo {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := make([]api.LocalWorkerInfo, 0, len(p.workers))
	for _, entry := range p.workers {
		if actual := entry.GetActual(); actual != nil {
			result = append(result, *actual)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Id < result[j].Id
	})
	return result
}

// GetAllWorkers fetches the last known state for all local workers.
func (p *localWorkerPool) GetAllWorkers() []api.LocalWorker {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := make([]api.LocalWorker, 0, len(p.workers))
	for _, entry := range p.workers {
		result = append(result, entry.LocalWorker)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Id < result[j].Id
	})
	return result
}

// SetRequest sets the requested state of a local worker
func (p *localWorkerPool) SetRequest(ctx context.Context, lw api.LocalWorker) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	id := lw.GetId()
	entry, found := p.workers[id]
	if !found {
		entry = &localWorkerEntry{}
		entry.LocalWorker.Id = id
		p.workers[id] = entry
	}
	entry.LocalWorker.Request = lw.GetRequest().Clone()
	// Do not change last updated at
	p.actuals.Pub(entry.LocalWorker)
	return nil
}

// SetActual sets the actual state of a local worker
func (p *localWorkerPool) SetActual(ctx context.Context, lw api.LocalWorker) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	id := lw.GetId()
	entry, found := p.workers[id]
	if !found {
		entry = &localWorkerEntry{}
		entry.LocalWorker.Id = id
		p.workers[id] = entry
	}
	entry.LocalWorker.Actual = lw.GetActual().Clone()
	entry.lastUpdatedActualAt = time.Now()
	p.actuals.Pub(entry.LocalWorker)
	return nil
}

// SubRequests is used to subscribe to all request changes of local workers.
func (p *localWorkerPool) SubRequests(enabled bool) (chan api.LocalWorker, context.CancelFunc) {
	c := make(chan api.LocalWorker)
	if enabled {
		cb := func(msg api.LocalWorker) {
			c <- msg
		}
		p.requests.Sub(cb)
		// Push all
		for _, lw := range p.GetAllWorkers() {
			p.requests.Pub(lw)
		}
		return c, func() {
			p.requests.Leave(cb)
			close(c)
		}
	} else {
		return c, func() {
			close(c)
		}
	}
}

// SubActuals is used to subscribe to actual changes of local workers.
func (p *localWorkerPool) SubActuals(enabled bool) (chan api.LocalWorker, context.CancelFunc) {
	c := make(chan api.LocalWorker)
	if enabled {
		cb := func(msg api.LocalWorker) {
			c <- msg
		}
		p.actuals.Sub(cb)
		return c, func() {
			p.actuals.Leave(cb)
			close(c)
		}
	} else {
		return c, func() {
			close(c)
		}
	}
}
