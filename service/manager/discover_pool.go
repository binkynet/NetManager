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
	"math/rand"
	"sync"
	"time"

	api "github.com/binkynet/BinkyNet/apis/v1"
	"github.com/mattn/go-pubsub"
	"github.com/rs/zerolog"
)

type discoverPool struct {
	log       zerolog.Logger
	mutex     sync.RWMutex
	clock     api.Clock
	requests  *pubsub.PubSub
	responses *pubsub.PubSub
}

func newDiscoverPool(log zerolog.Logger) *discoverPool {
	return &discoverPool{
		log:       log.With().Str("pool", "discovery").Logger(),
		requests:  pubsub.New(),
		responses: pubsub.New(),
	}
}

type discoverRequest struct {
	id        string
	requestID int32
}

// Trigger a discovery and wait for the response.
func (p *discoverPool) Trigger(ctx context.Context, id string) (*api.DiscoverResult, error) {
	// Subscribe to results
	timeout := getTimeout(ctx, time.Minute)
	resultChan, cancel := p.SubActuals(true, timeout, id)
	defer cancel()

	// Trigger discover
	p.log.Debug().Str("id", id).Msg("discoverPool.Pub")
	requestID := rand.Int31()
	p.SetDiscoverRequest(api.DeviceDiscovery{
		Id: id,
		Request: &api.DiscoverRequest{
			RequestId: requestID,
		},
	})
	p.requests.Pub(&discoverRequest{id: id})

	// Wait for response
	for {
		select {
		case msg := <-resultChan:
			p.log.Debug().Str("id", msg.GetId()).Msg("Received result")
			if msg.GetId() == id {
				return msg.GetActual(), nil
			}
		case <-ctx.Done():
			// Context canceled
			return nil, ctx.Err()
		}
	}
}

// SubRequests is called by the LocalWorker GRPC API to wait for discover requests.
func (p *discoverPool) SubRequests(enabled bool, timeout time.Duration, id string) (chan api.DeviceDiscovery, context.CancelFunc) {
	discoverPoolMetrics.SubRequestTotalCounter.Inc()
	c := make(chan api.DeviceDiscovery)
	if enabled {
		cb := func(msg api.DeviceDiscovery) {
			if id == "" || msg.GetId() == id {
				select {
				case c <- msg:
					// Done
					discoverPoolMetrics.SubRequestMessagesTotalCounters.WithLabelValues(id).Inc()
				case <-time.After(timeout):
					p.log.Error().
						Dur("timeout", timeout).
						Msg("Failed to deliver DeviceDiscovery request to channel")
					discoverPoolMetrics.SubRequestMessagesFailedTotalCounters.WithLabelValues(id).Inc()
				}
			} else {
				p.log.Debug().Str("msg_id", msg.GetId()).Msg("Skipping message for other localworker")
			}
		}
		p.requests.Sub(cb)
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

// subResponse is called by Trigger.
func (p *discoverPool) SubActuals(enabled bool, timeout time.Duration, id string) (chan api.DeviceDiscovery, context.CancelFunc) {
	discoverPoolMetrics.SubActualTotalCounter.Inc()
	c := make(chan api.DeviceDiscovery)
	if enabled {
		cb := func(msg api.DeviceDiscovery) {
			if id == "" || msg.GetId() == id {
				select {
				case c <- msg:
					// Done
					discoverPoolMetrics.SubActualMessagesTotalCounters.WithLabelValues(id).Inc()
				case <-time.After(timeout):
					p.log.Error().
						Dur("timeout", timeout).
						Msg("Failed to deliver DeviceDiscovery actual to channel")
					discoverPoolMetrics.SubActualMessagesFailedTotalCounters.WithLabelValues(id).Inc()
				}
			}
		}
		p.responses.Sub(cb)
		return c, func() {
			p.responses.Leave(cb)
			close(c)
		}
	} else {
		return c, func() {
			close(c)
		}
	}
}

// SetDiscoverRequest triggers a discovery request
func (p *discoverPool) SetDiscoverRequest(req api.DeviceDiscovery) {
	safePub(p.log, p.requests, req)
	discoverPoolMetrics.SetRequestTotalCounters.WithLabelValues(req.GetId()).Inc()
}

// SetDiscoverResult is called by the local worker in response to discover requests.
func (p *discoverPool) SetDiscoverResult(req api.DeviceDiscovery) error {
	safePub(p.log, p.responses, req)
	discoverPoolMetrics.SetActualTotalCounters.WithLabelValues(req.GetId()).Inc()
	return nil
}
