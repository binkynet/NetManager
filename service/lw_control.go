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
	"fmt"

	"github.com/binkynet/BinkyNet/apis/util"
	api "github.com/binkynet/BinkyNet/apis/v1"
)

// Ping messages are send at regular intervals by local workers
// as a heartbeat notification, as well as providing information about
// versions.
func (s *service) Ping(ctx context.Context, req *api.LocalWorkerInfo) (*api.Empty, error) {
	if req != nil {
		s.Manager.SetLocalWorkerUpdate(ctx, *req)
	}
	return &api.Empty{}, nil
}

// GetDiscoverRequests is used to allow the netmanager to request a discovery by
// the local worker.
// The local worker in turn responds with a SetDiscoverResult call.
func (s *service) GetDiscoverRequests(req *api.LocalWorkerInfo, server api.LocalWorkerControlService_GetDiscoverRequestsServer) error {
	ch, cancel := s.Manager.SubscribeDiscoverRequests(req.GetId())
	log := s.Log.With().Str("id", req.GetId()).Logger()
	log.Debug().Msg("GetDiscoverRequests...")
	defer cancel()
	ctx := server.Context()
	for {
		select {
		case msg := <-ch:
			log.Debug().Msg("Sending DiscoverRequest...")
			if err := server.Send(&msg); err != nil {
				return err
			}
		case <-ctx.Done():
			// Context canceled
			log.Debug().Msg("Canceling DiscoverRequest...")
			return nil
		}
	}
}

// SetDiscoverResult is called by the local worker in response to discover requests.
func (s *service) SetDiscoverResult(ctx context.Context, req *api.DiscoverResult) (*api.Empty, error) {
	s.Log.Debug().
		Strs("addrs", req.GetAddresses()).
		Str("id", req.GetId()).
		Msg("SetDiscoverResult...")
	if req == nil {
		return nil, fmt.Errorf("Missing result")
	}
	if err := s.Manager.SetDiscoverResult(ctx, *req); err != nil {
		return nil, err
	}
	return &api.Empty{}, nil
}

// GetPowerRequests is used to get a stream of power requests from the network
// master.
func (s *service) GetPowerRequests(req *api.PowerRequestsOptions, server api.LocalWorkerControlService_GetPowerRequestsServer) error {
	ch, cancel := s.Manager.SubscribePowerRequests()
	defer cancel()
	ctx := server.Context()
	for {
		select {
		case msg := <-ch:
			if err := server.Send(msg.GetRequest()); err != nil {
				return err
			}
		case <-ctx.Done():
			// Context canceled
			return nil
		}
	}
}

// SetPowerActuals is used to send a stream of actual power statuses to
// the network master.
func (s *service) SetPowerActuals(server api.LocalWorkerControlService_SetPowerActualsServer) error {
	for {
		msg, err := server.Recv()
		if util.IsStreamClosed(err) {
			return nil
		} else if err != nil {
			return err
		}
		s.Manager.SetPowerActual(*msg)
	}
}

// GetLocRequests is used to get a stream of loc requests from the network
// master.
func (s *service) GetLocRequests(req *api.LocRequestsOptions, server api.LocalWorkerControlService_GetLocRequestsServer) error {
	ch, cancel := s.Manager.SubscribeLocRequests()
	defer cancel()
	ctx := server.Context()
	for {
		select {
		case msg := <-ch:
			if err := server.Send(&msg); err != nil {
				return err
			}
		case <-ctx.Done():
			// Context canceled
			return nil
		}
	}
}

// SetLocActuals is used to send a stream of actual loc statuses to
// the network master.
func (s *service) SetLocActuals(server api.LocalWorkerControlService_SetLocActualsServer) error {
	for {
		msg, err := server.Recv()
		if util.IsStreamClosed(err) {
			return nil
		} else if err != nil {
			return err
		}
		s.Manager.SetLocActual(*msg)
	}
}

// SetSensorActuals is used to send a stream of actual sensor statuses to
// the network master.
func (s *service) SetSensorActuals(server api.LocalWorkerControlService_SetSensorActualsServer) error {
	for {
		msg, err := server.Recv()
		if util.IsStreamClosed(err) {
			return nil
		} else if err != nil {
			return err
		}
		s.Manager.SetSensorActual(*msg)
	}
}

// GetOutputRequests is used to get a stream of output requests from the network
// master.
func (s *service) GetOutputRequests(req *api.OutputRequestsOptions, server api.LocalWorkerControlService_GetOutputRequestsServer) error {
	ch, cancel := s.Manager.SubscribeOutputRequests()
	defer cancel()
	ctx := server.Context()
	for {
		select {
		case msg := <-ch:
			if err := server.Send(&msg); err != nil {
				return err
			}
		case <-ctx.Done():
			// Context canceled
			return nil
		}
	}
}

// SetOutputActuals is used to send a stream of actual output statuses to
// the network master.
func (s *service) SetOutputActuals(server api.LocalWorkerControlService_SetOutputActualsServer) error {
	for {
		msg, err := server.Recv()
		if util.IsStreamClosed(err) {
			return nil
		} else if err != nil {
			return err
		}
		s.Manager.SetOutputActual(*msg)
	}
}

// GetSwitchRequests is used to get a stream of switch requests from the network
// master.
func (s *service) GetSwitchRequests(req *api.SwitchRequestsOptions, server api.LocalWorkerControlService_GetSwitchRequestsServer) error {
	ch, cancel := s.Manager.SubscribeSwitchRequests()
	defer cancel()
	ctx := server.Context()
	for {
		select {
		case msg := <-ch:
			if err := server.Send(&msg); err != nil {
				return err
			}
		case <-ctx.Done():
			// Context canceled
			return nil
		}
	}
}

// SetSwitchActuals is used to send a stream of actual switch statuses to
// the network master.
func (s *service) SetSwitchActuals(server api.LocalWorkerControlService_SetSwitchActualsServer) error {
	for {
		msg, err := server.Recv()
		if util.IsStreamClosed(err) {
			return nil
		} else if err != nil {
			return err
		}
		s.Manager.SetSwitchActual(*msg)
	}
}

// GetClock is used to get a stream of switch current time of day from the network
// master.
func (s *service) GetClock(req *api.Empty, server api.LocalWorkerControlService_GetClockServer) error {
	ch, cancel := s.Manager.SubscribeClockActuals()
	defer cancel()
	ctx := server.Context()
	for {
		select {
		case msg := <-ch:
			if err := server.Send(&msg); err != nil {
				return err
			}
		case <-ctx.Done():
			// Context canceled
			return nil
		}
	}
}
