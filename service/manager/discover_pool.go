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
	"sync"

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
		log:       log,
		requests:  pubsub.New(),
		responses: pubsub.New(),
	}
}

type discoverRequest struct {
	id string
}

// Trigger a discovery and wait for the response.
func (p *discoverPool) Trigger(ctx context.Context, id string) (*api.DiscoverResult, error) {
	// Subscribe to results
	resultChan, cancel := p.subResponse()
	defer cancel()

	// Trigger discover
	p.log.Debug().Str("id", id).Msg("discoverPool.Pub")
	p.requests.Pub(&discoverRequest{id: id})

	// Wait for response
	for {
		select {
		case msg := <-resultChan:
			p.log.Debug().Str("id", msg.GetId()).Msg("Received result")
			if msg.GetId() == id {
				return &msg, nil
			}
		case <-ctx.Done():
			// Context canceled
			return nil, ctx.Err()
		}
	}
}

// SubRequest is called by the LocalWorker GRPC API to wait for discover requests.
func (p *discoverPool) SubRequest(id string) (chan api.DiscoverRequest, context.CancelFunc) {
	c := make(chan api.DiscoverRequest)
	cb := func(msg *discoverRequest) {
		if msg.id == id {
			c <- api.DiscoverRequest{}
		} else {
			p.log.Debug().Str("msg_id", msg.id).Msg("Skipping message for other localworker")
		}
	}
	p.requests.Sub(cb)
	return c, func() {
		p.requests.Leave(cb)
		close(c)
	}
}

// subResponse is called by Trigger.
func (p *discoverPool) subResponse() (chan api.DiscoverResult, context.CancelFunc) {
	c := make(chan api.DiscoverResult)
	cb := func(msg api.DiscoverResult) {
		c <- msg
	}
	p.responses.Sub(cb)
	return c, func() {
		p.responses.Leave(cb)
		close(c)
	}
}

// SetDiscoverResult is called by the local worker in response to discover requests.
func (p *discoverPool) SetDiscoverResult(req api.DiscoverResult) error {
	p.responses.Pub(req)
	return nil
}
