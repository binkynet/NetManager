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
	return nil // TODO
}

// SetPowerActuals is used to send a stream of actual power statuses to
// the network master.
func (s *service) SetPowerActuals(server api.LocalWorkerControlService_SetPowerActualsServer) error {
	return nil // TODO
}

// GetLocRequests is used to get a stream of loc requests from the network
// master.
func (s *service) GetLocRequests(req *api.LocRequestsOptions, server api.LocalWorkerControlService_GetLocRequestsServer) error {
	return nil // TODO
}

// SetLocActuals is used to send a stream of actual loc statuses to
// the network master.
func (s *service) SetLocActuals(server api.LocalWorkerControlService_SetLocActualsServer) error {
	return nil // TODO
}
