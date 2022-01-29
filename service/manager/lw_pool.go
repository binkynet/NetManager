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
	log     zerolog.Logger
	mutex   sync.RWMutex
	updates *pubsub.PubSub
	workers map[string]localWorkerEntry
}

type localWorkerEntry struct {
	api.LocalWorkerInfo
	lastUpdatedAt time.Time
}

func newLocalWorkerPool(log zerolog.Logger) *localWorkerPool {
	return &localWorkerPool{
		log:     log,
		updates: pubsub.New(),
		workers: make(map[string]localWorkerEntry),
	}
}

// GetInfo fetches the last known info for a local worker with given ID.
// Returns: info, lastUpdatedAt, found
func (p *localWorkerPool) GetInfo(id string) (api.LocalWorkerInfo, time.Time, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	info, found := p.workers[id]
	return info.LocalWorkerInfo, info.lastUpdatedAt, found
}

// GetAll fetches the last known info for all local workers.
func (p *localWorkerPool) GetAll() []api.LocalWorkerInfo {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := make([]api.LocalWorkerInfo, 0, len(p.workers))
	for _, info := range p.workers {
		result = append(result, info.LocalWorkerInfo)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Id < result[j].Id
	})
	return result
}

// SetUpdate informs the pool and its listeners of a local worker update.
func (p *localWorkerPool) SetUpdate(ctx context.Context, info api.LocalWorkerInfo) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.workers[info.GetId()] = localWorkerEntry{
		LocalWorkerInfo: info,
		lastUpdatedAt:   time.Now(),
	}
	p.updates.Pub(info)
}

// SubUpdates is used to subscribe to updates of all local workers.
func (p *localWorkerPool) SubUpdates() (chan api.LocalWorkerInfo, context.CancelFunc) {
	c := make(chan api.LocalWorkerInfo)
	cb := func(msg api.LocalWorkerInfo) {
		c <- msg
	}
	p.updates.Sub(cb)
	return c, func() {
		p.updates.Leave(cb)
		close(c)
	}
}
