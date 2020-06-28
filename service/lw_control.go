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

	api "github.com/binkynet/BinkyNet/apis/v1"
)

// Ping messages are send at regular intervals by local workers
// as a heartbeat notification, as well as providing information about
// versions.
func (s *service) Ping(ctx context.Context, req *api.LocalWorkerInfo) (*api.Empty, error) {
	return &api.Empty{}, nil // TODO
}

// GetPowerRequests is used to get a stream of power requests from the network
// master.
func (s *service) GetPowerRequests(req *api.PowerRequestsOptions, server api.LocalWorkerControlService_GetPowerRequestsServer) error {
	ch, cancel := s.powerPool.SubRequest()
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
		if isStreamClosed(err) {
			return nil
		} else if err != nil {
			return err
		}
		s.powerPool.SetActual(*msg)
	}
}

// GetLocRequests is used to get a stream of loc requests from the network
// master.
func (s *service) GetLocRequests(req *api.LocRequestsOptions, server api.LocalWorkerControlService_GetLocRequestsServer) error {
	ch, cancel := s.locPool.SubRequest()
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
		if isStreamClosed(err) {
			return nil
		} else if err != nil {
			return err
		}
		s.locPool.SetActual(*msg)
	}
}

// SetSensorActuals is used to send a stream of actual sensor statuses to
// the network master.
func (s *service) SetSensorActuals(server api.LocalWorkerControlService_SetSensorActualsServer) error {
	for {
		msg, err := server.Recv()
		if isStreamClosed(err) {
			return nil
		} else if err != nil {
			return err
		}
		s.sensorPool.SetActual(*msg)
	}
}

// GetOutputRequests is used to get a stream of output requests from the network
// master.
func (s *service) GetOutputRequests(req *api.OutputRequestsOptions, server api.LocalWorkerControlService_GetOutputRequestsServer) error {
	ch, cancel := s.outputPool.SubRequest()
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
		if isStreamClosed(err) {
			return nil
		} else if err != nil {
			return err
		}
		s.outputPool.SetActual(*msg)
	}
}

// GetSwitchRequests is used to get a stream of switch requests from the network
// master.
func (s *service) GetSwitchRequests(req *api.SwitchRequestsOptions, server api.LocalWorkerControlService_GetSwitchRequestsServer) error {
	ch, cancel := s.switchPool.SubRequest()
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
		if isStreamClosed(err) {
			return nil
		} else if err != nil {
			return err
		}
		s.switchPool.SetActual(*msg)
	}
}

// GetClock is used to get a stream of switch current time of day from the network
// master.
func (s *service) GetClock(req *api.Empty, server api.LocalWorkerControlService_GetClockServer) error {
	ch, cancel := s.clockPool.SubActual()
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
