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
	"crypto/rand"
	"fmt"
	"sort"
	"sync"
	"time"

	api "github.com/binkynet/BinkyNet/apis/v1"
	"github.com/binkynet/NetManager/service/util"
	"github.com/mattn/go-pubsub"
	"github.com/rs/zerolog"
)

type localWorkerPool struct {
	log        zerolog.Logger
	mutex      sync.RWMutex
	requests   *pubsub.PubSub
	actuals    *pubsub.PubSub
	workers    map[string]*localWorkerEntry
	hashPrefix string
}

type localWorkerEntry struct {
	remoteAddr string
	api.LocalWorker
	lastUpdatedActualAt time.Time
}

func newLocalWorkerPool(log zerolog.Logger) *localWorkerPool {
	rndData := make([]byte, 4)
	rand.Read(rndData)
	return &localWorkerPool{
		log:        log,
		requests:   pubsub.New(),
		actuals:    pubsub.New(),
		workers:    make(map[string]*localWorkerEntry),
		hashPrefix: fmt.Sprintf("%x", rndData),
	}
}

// GetInfo fetches the last known info for a local worker with given ID.
// Returns: info, remoteAddr, lastUpdatedAt, found
func (p *localWorkerPool) GetInfo(id string) (api.LocalWorkerInfo, string, time.Time, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if lw, found := p.workers[id]; found {
		if info := lw.GetActual(); info != nil {
			return *info, lw.remoteAddr, lw.lastUpdatedActualAt, found
		}
	}
	return api.LocalWorkerInfo{}, "", time.Time{}, false
}

// GetLocalWorkerServiceClient constructs a client to the LocalWorkerService served
// on the local worker with given ID.
func (p *localWorkerPool) GetLocalWorkerServiceClient(id string) (api.LocalWorkerServiceClient, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if lw, found := p.workers[id]; found {
		port := 0
		secure := false
		if info := lw.GetActual(); info != nil {
			port = int(info.GetLocalWorkerServicePort())
			secure = info.GetLocalWorkerServiceSecure()
		}
		if port == 0 {
			return nil, fmt.Errorf("local worker [%s] does not provide local worker service port", id)
		}
		conn, err := util.DialConn(lw.remoteAddr, port, secure)
		if err != nil {
			return nil, fmt.Errorf("failed to dial local worker: %w", err)
		}
		return api.NewLocalWorkerServiceClient(conn), nil
	}
	return nil, fmt.Errorf("local worker [%s] not found", id)
}

// RequestReset requests the local worker with given ID to reset itself.
func (p *localWorkerPool) RequestReset(ctx context.Context, id string) error {
	client, err := p.GetLocalWorkerServiceClient(id)
	if err != nil {
		return fmt.Errorf("failed to create local worker service client: %w", err)
	}
	if _, err := client.Reset(ctx, &api.Empty{}); err != nil {
		return fmt.Errorf("failed to request reset of local worker: %w", err)
	}
	return nil
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
	lwPoolMetrics.SetRequestTotalCounters.WithLabelValues(lw.GetId()).Inc()
	req := lw.GetRequest()
	if req == nil {
		return fmt.Errorf("Request missing")
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()

	id := lw.GetId()
	entry, found := p.workers[id]
	if !found {
		entry = &localWorkerEntry{}
		entry.LocalWorker.Id = id
		p.workers[id] = entry
	}
	entry.LocalWorker.Request = req.Clone()
	// Set hash
	entry.LocalWorker.Request.Hash = p.hashPrefix + req.Sha1()
	// Do not change last updated at
	safePub(p.log, p.requests, entry.LocalWorker)
	return nil
}

// SetActual sets the actual state of a local worker
func (p *localWorkerPool) SetActual(ctx context.Context, lw api.LocalWorker, remoteAddr string) error {
	lwPoolMetrics.SetActualTotalCounters.WithLabelValues(lw.GetId()).Inc()
	p.mutex.Lock()
	defer p.mutex.Unlock()

	id := lw.GetId()
	entry, found := p.workers[id]
	if !found {
		entry = &localWorkerEntry{}
		entry.LocalWorker.Id = id
		p.workers[id] = entry
	}
	entry.remoteAddr = remoteAddr
	entry.LocalWorker.Actual = lw.GetActual().Clone()
	entry.lastUpdatedActualAt = time.Now()
	safePub(p.log, p.actuals, entry.LocalWorker)
	return nil
}

// SubRequests is used to subscribe to all request changes of local workers.
func (p *localWorkerPool) SubRequests(enabled bool, timeout time.Duration, filter ModuleFilter) (chan api.LocalWorker, context.CancelFunc) {
	lwPoolMetrics.SubRequestTotalCounter.Inc()
	c := make(chan api.LocalWorker)
	if enabled {
		// Subscribe
		cb := func(msg api.LocalWorker) {
			if msg.Request != nil && filter.MatchesModuleID(msg.GetId()) {
				msg.Request.Unixtime = time.Now().Unix()
				select {
				case c <- msg:
					// Done
					lwPoolMetrics.SubRequestMessagesTotalCounters.WithLabelValues(msg.GetId()).Inc()
				case <-time.After(timeout):
					p.log.Error().
						Dur("timeout", timeout).
						Msg("Failed to deliver local worker request to channel")
					lwPoolMetrics.SubRequestMessagesFailedTotalCounters.WithLabelValues(msg.GetId()).Inc()
				}
			}
		}
		p.requests.Sub(cb)
		// Push all known request states
		for _, lw := range p.GetAllWorkers() {
			if lw.GetRequest() != nil && filter.MatchesModuleID(lw.GetId()) {
				go cb(lw)
			}
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
func (p *localWorkerPool) SubActuals(enabled bool, timeout time.Duration, filter ModuleFilter) (chan api.LocalWorker, context.CancelFunc) {
	lwPoolMetrics.SubActualTotalCounter.Inc()
	c := make(chan api.LocalWorker)
	if enabled {
		cb := func(msg api.LocalWorker) {
			if filter.MatchesModuleID(msg.GetId()) {
				if msg.Request != nil {
					msg.Request.Unixtime = time.Now().Unix()
				}
				select {
				case c <- msg:
					// Done
					lwPoolMetrics.SubActualMessagesTotalCounters.WithLabelValues(msg.GetId()).Inc()
				case <-time.After(timeout):
					p.log.Error().
						Dur("timeout", timeout).
						Msg("Failed to deliver local worker actual to channel")
					lwPoolMetrics.SubActualMessagesFailedTotalCounters.WithLabelValues(msg.GetId()).Inc()
				}
			}
		}
		p.actuals.Sub(cb)
		// Push all known actual states
		for _, lw := range p.GetAllWorkers() {
			if lw.GetActual() != nil && filter.MatchesModuleID(lw.GetId()) {
				go cb(lw)
			}
		}
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
