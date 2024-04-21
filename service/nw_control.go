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
	"net"
	"time"

	api "github.com/binkynet/BinkyNet/apis/v1"
	"github.com/binkynet/NetManager/service/manager"
	"google.golang.org/grpc/peer"
)

const (
	chanTimeout = time.Second * 5
)

// Set the requested local worker state
func (s *service) SetLocalWorkerRequest(ctx context.Context, req *api.LocalWorker) (*api.Empty, error) {
	lwMetrics.SetRequestTotalCounters.WithLabelValues(req.GetId()).Inc()
	if err := s.Manager.SetLocalWorkerRequest(ctx, *req); err != nil {
		return nil, err
	}
	return &api.Empty{}, nil
}

// Set the actual local worker state
func (s *service) SetLocalWorkerActual(ctx context.Context, req *api.LocalWorker) (*api.Empty, error) {
	lwMetrics.SetActualTotalCounters.WithLabelValues(req.GetId()).Inc()
	var remoteAddr string
	if pr, ok := peer.FromContext(ctx); ok {
		remoteAddr = pr.Addr.String()
		remoteAddr, _, _ = net.SplitHostPort(remoteAddr)
	}
	if err := s.Manager.SetLocalWorkerActual(ctx, *req, remoteAddr); err != nil {
		return nil, err
	}
	return &api.Empty{}, nil
}

// Watch local worker changes
func (s *service) WatchLocalWorkers(req *api.WatchOptions, server api.NetworkControlService_WatchLocalWorkersServer) error {
	lwMetrics.WatchTotalCounter.Inc()
	ctx := server.Context()
	ach, acancel := s.Manager.SubscribeLocalWorkerActuals(req.GetWatchActualChanges(), chanTimeout, manager.ModuleFilter(req.GetModuleId()))
	defer acancel()
	rch, rcancel := s.Manager.SubscribeLocalWorkerRequests(req.GetWatchRequestChanges(), chanTimeout, manager.ModuleFilter(req.GetModuleId()))
	defer rcancel()
	for {
		select {
		case msg := <-ach:
			if err := server.Send(&msg); err != nil {
				s.Log.Warn().Err(err).Msg("Send local worker actual failed")
				lwMetrics.WatchActualMessagesFailedTotalCounters.WithLabelValues(msg.GetId()).Inc()
				return err
			}
			lwMetrics.WatchActualMessagesTotalCounters.WithLabelValues(msg.GetId()).Inc()
		case msg := <-rch:
			if err := server.Send(&msg); err != nil {
				s.Log.Warn().Err(err).Msg("Send local worker request failed")
				lwMetrics.WatchRequestMessagesFailedTotalCounters.WithLabelValues(msg.GetId()).Inc()
				return err
			}
			lwMetrics.WatchRequestMessagesTotalCounters.WithLabelValues(msg.GetId()).Inc()
		case <-ctx.Done():
			// Context canceled
			return nil
		}
	}
}

// Set an actual device discovery state
func (s *service) SetDeviceDiscoveryActual(ctx context.Context, req *api.DeviceDiscovery) (*api.Empty, error) {
	discoverMetrics.SetActualTotalCounters.WithLabelValues(req.GetId()).Inc()
	if err := s.Manager.SetDevicesDiscoveryActual(ctx, *req); err != nil {
		return nil, err
	}
	return &api.Empty{}, nil
}

func (s *service) SetPowerActual(ctx context.Context, req *api.PowerState) (*api.Empty, error) {
	powerMetrics.SetActualTotalCounters.WithLabelValues("power").Inc()
	s.Manager.SetPowerActual(*req)
	return &api.Empty{}, nil
}

func (s *service) SetLocActual(ctx context.Context, req *api.Loc) (*api.Empty, error) {
	locMetrics.SetActualTotalCounters.WithLabelValues(string(req.GetAddress())).Inc()
	s.Manager.SetLocActual(*req)
	return &api.Empty{}, nil
}

func (s *service) SetSensorActual(ctx context.Context, req *api.Sensor) (*api.Empty, error) {
	sensorMetrics.SetActualTotalCounters.WithLabelValues(string(req.GetAddress())).Inc()
	s.Manager.SetSensorActual(*req)
	return &api.Empty{}, nil
}

func (s *service) SetOutputActual(ctx context.Context, req *api.Output) (*api.Empty, error) {
	outputMetrics.SetActualTotalCounters.WithLabelValues(string(req.GetAddress())).Inc()
	s.Manager.SetOutputActual(*req)
	return &api.Empty{}, nil
}

func (s *service) SetSwitchActual(ctx context.Context, req *api.Switch) (*api.Empty, error) {
	switchMetrics.SetActualTotalCounters.WithLabelValues(string(req.GetAddress())).Inc()
	s.Manager.SetSwitchActual(*req)
	return &api.Empty{}, nil
}

// Set an actual clock state
func (s *service) SetClockActual(ctx context.Context, req *api.Clock) (*api.Empty, error) {
	clockMetrics.SetActualTotalCounters.WithLabelValues("clock").Inc()
	s.Manager.SetClockActual(*req)
	return &api.Empty{}, nil
}

// Watch clock changes
func (s *service) WatchClock(req *api.WatchOptions, server api.NetworkControlService_WatchClockServer) error {
	clockMetrics.WatchTotalCounter.Inc()
	ctx := server.Context()
	ach, acancel := s.Manager.SubscribeClockActuals(req.GetWatchActualChanges(), chanTimeout)
	defer acancel()
	for {
		select {
		case msg := <-ach:
			if err := server.Send(&msg); err != nil {
				s.Log.Warn().Err(err).Msg("Send clock actual failed")
				clockMetrics.WatchActualMessagesFailedTotalCounters.WithLabelValues("clock").Inc()
				return err
			}
			clockMetrics.WatchActualMessagesTotalCounters.WithLabelValues("clock").Inc()
		case <-ctx.Done():
			// Context canceled
			return nil
		}
	}
}
