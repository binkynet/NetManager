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

	"github.com/binkynet/BinkyNet/apis/util"
	api "github.com/binkynet/BinkyNet/apis/v1"
	"github.com/binkynet/NetManager/service/manager"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/peer"
)

const (
	chanTimeout = time.Second * 5
)

// Set the requested local worker state
func (s *service) SetLocalWorkerRequest(ctx context.Context, req *api.LocalWorker) (*api.Empty, error) {
	if err := s.Manager.SetLocalWorkerRequest(ctx, *req); err != nil {
		return nil, err
	}
	return &api.Empty{}, nil
}

// Set the actual local worker state
func (s *service) SetLocalWorkerActual(ctx context.Context, req *api.LocalWorker) (*api.Empty, error) {
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
				return err
			}
		case msg := <-rch:
			if err := server.Send(&msg); err != nil {
				s.Log.Warn().Err(err).Msg("Send local worker request failed")
				return err
			}
		case <-ctx.Done():
			// Context canceled
			return nil
		}
	}
}

// Set a requested device discovery state
func (s *service) SetDeviceDiscoveryRequest(ctx context.Context, req *api.DeviceDiscovery) (*api.Empty, error) {
	if err := s.Manager.SetDevicesDiscoveryRequest(ctx, *req); err != nil {
		return nil, err
	}
	return &api.Empty{}, nil
}

// Set an actual device discovery state
func (s *service) SetDeviceDiscoveryActual(ctx context.Context, req *api.DeviceDiscovery) (*api.Empty, error) {
	if err := s.Manager.SetDevicesDiscoveryActual(ctx, *req); err != nil {
		return nil, err
	}
	return &api.Empty{}, nil
}

// Watch device discovery changes
func (s *service) WatchDeviceDiscoveries(req *api.WatchOptions, server api.NetworkControlService_WatchDeviceDiscoveriesServer) error {
	ctx := server.Context()
	ach, acancel := s.Manager.SubscribeDiscoverRequests(req.GetWatchActualChanges(), chanTimeout, req.GetModuleId())
	defer acancel()
	rch, rcancel := s.Manager.SubscribeDiscoverActuals(req.GetWatchRequestChanges(), chanTimeout, req.GetModuleId())
	defer rcancel()
	for {
		select {
		case msg := <-ach:
			if err := server.Send(&msg); err != nil {
				s.Log.Warn().Err(err).Msg("Send device discovery actual failed")
				return err
			}
		case msg := <-rch:
			if err := server.Send(&msg); err != nil {
				s.Log.Warn().Err(err).Msg("Send device discovery request failed")
				return err
			}
		case <-ctx.Done():
			// Context canceled
			return nil
		}
	}

}

func (s *service) SetPowerRequest(ctx context.Context, req *api.PowerState) (*api.Empty, error) {
	s.Manager.SetPowerRequest(*req)
	return &api.Empty{}, nil
}

func (s *service) SetPowerActual(ctx context.Context, req *api.PowerState) (*api.Empty, error) {
	s.Manager.SetPowerActual(*req)
	return &api.Empty{}, nil
}

func (s *service) WatchPower(req *api.WatchOptions, server api.NetworkControlService_WatchPowerServer) error {
	ctx := server.Context()
	ach, acancel := s.Manager.SubscribePowerActuals(req.GetWatchActualChanges(), chanTimeout)
	defer acancel()
	rch, rcancel := s.Manager.SubscribePowerRequests(req.GetWatchRequestChanges(), chanTimeout)
	defer rcancel()
	for {
		select {
		case msg := <-ach:
			if err := server.Send(&msg); err != nil {
				s.Log.Warn().Err(err).Msg("Send power actual failed")
				return err
			}
		case msg := <-rch:
			if err := server.Send(&msg); err != nil {
				s.Log.Warn().Err(err).Msg("Send power request failed")
				return err
			}
		case <-ctx.Done():
			// Context canceled
			return nil
		}
	}
}

func (s *service) SetLocRequest(ctx context.Context, req *api.Loc) (*api.Empty, error) {
	s.Manager.SetLocRequest(*req)
	return &api.Empty{}, nil
}

func (s *service) SetLocActual(ctx context.Context, req *api.Loc) (*api.Empty, error) {
	s.Manager.SetLocActual(*req)
	return &api.Empty{}, nil
}

func (s *service) WatchLocs(req *api.WatchOptions, server api.NetworkControlService_WatchLocsServer) error {
	ctx := server.Context()
	ach, acancel := s.Manager.SubscribeLocActuals(req.GetWatchActualChanges(), chanTimeout)
	defer acancel()
	rch, rcancel := s.Manager.SubscribeLocRequests(req.GetWatchRequestChanges(), chanTimeout)
	defer rcancel()
	for {
		select {
		case msg := <-ach:
			if err := server.Send(&msg); err != nil {
				s.Log.Warn().Err(err).Msg("Send loc actual failed")
				return err
			}
		case msg := <-rch:
			if err := server.Send(&msg); err != nil {
				s.Log.Warn().Err(err).Msg("Send loc request failed")
				return err
			}
		case <-ctx.Done():
			// Context canceled
			return nil
		}
	}
}

func (s *service) SetSensorActual(ctx context.Context, req *api.Sensor) (*api.Empty, error) {
	s.Manager.SetSensorActual(*req)
	return &api.Empty{}, nil
}

func (s *service) WatchSensors(req *api.WatchOptions, server api.NetworkControlService_WatchSensorsServer) error {
	ctx := server.Context()
	ach, acancel := s.Manager.SubscribeSensorActuals(req.GetWatchActualChanges(), chanTimeout, manager.ModuleFilter(req.GetModuleId()))
	defer acancel()
	for {
		select {
		case msg := <-ach:
			if err := server.Send(&msg); err != nil {
				s.Log.Warn().Err(err).Msg("Send sensor actual failed")
				return err
			}
		case <-ctx.Done():
			// Context canceled
			return nil
		}
	}
}

func (s *service) SetOutputRequest(ctx context.Context, req *api.Output) (*api.Empty, error) {
	s.Manager.SetOutputRequest(*req)
	return &api.Empty{}, nil
}

func (s *service) SetOutputActual(ctx context.Context, req *api.Output) (*api.Empty, error) {
	s.Manager.SetOutputActual(*req)
	return &api.Empty{}, nil
}

func (s *service) WatchOutputs(req *api.WatchOptions, server api.NetworkControlService_WatchOutputsServer) error {
	ctx := server.Context()
	ach, acancel := s.Manager.SubscribeOutputActuals(req.GetWatchActualChanges(), chanTimeout, manager.ModuleFilter(req.GetModuleId()))
	defer acancel()
	rch, rcancel := s.Manager.SubscribeOutputRequests(req.GetWatchRequestChanges(), chanTimeout, manager.ModuleFilter(req.GetModuleId()))
	defer rcancel()
	for {
		select {
		case msg := <-ach:
			if err := server.Send(&msg); err != nil {
				s.Log.Warn().Err(err).Msg("Send output actual failed")
				return err
			}
		case msg := <-rch:
			if err := server.Send(&msg); err != nil {
				s.Log.Warn().Err(err).Msg("Send output request failed")
				return err
			}
		case <-ctx.Done():
			// Context canceled
			return nil
		}
	}
}

func (s *service) SetSwitchRequest(ctx context.Context, req *api.Switch) (*api.Empty, error) {
	s.Manager.SetSwitchRequest(*req)
	return &api.Empty{}, nil
}

func (s *service) SetSwitchActual(ctx context.Context, req *api.Switch) (*api.Empty, error) {
	s.Manager.SetSwitchActual(*req)
	return &api.Empty{}, nil
}

func (s *service) WatchSwitches(req *api.WatchOptions, server api.NetworkControlService_WatchSwitchesServer) error {
	ctx := server.Context()
	ach, acancel := s.Manager.SubscribeSwitchActuals(req.GetWatchActualChanges(), chanTimeout, manager.ModuleFilter(req.GetModuleId()))
	defer acancel()
	rch, rcancel := s.Manager.SubscribeSwitchRequests(req.GetWatchRequestChanges(), chanTimeout, manager.ModuleFilter(req.GetModuleId()))
	defer rcancel()
	for {
		select {
		case msg := <-ach:
			if err := server.Send(&msg); err != nil {
				s.Log.Warn().Err(err).Msg("Send switch actual failed")
				return err
			}
		case msg := <-rch:
			if err := server.Send(&msg); err != nil {
				s.Log.Warn().Err(err).Msg("Send request actual failed")
				return err
			}
		case <-ctx.Done():
			// Context canceled
			return nil
		}
	}
}

// Set an actual clock state
func (s *service) SetClockActual(ctx context.Context, req *api.Clock) (*api.Empty, error) {
	s.Manager.SetClockActual(*req)
	return &api.Empty{}, nil
}

// Watch clock changes
func (s *service) WatchClock(req *api.WatchOptions, server api.NetworkControlService_WatchClockServer) error {
	ctx := server.Context()
	ach, acancel := s.Manager.SubscribeClockActuals(req.GetWatchActualChanges(), chanTimeout)
	defer acancel()
	for {
		select {
		case msg := <-ach:
			if err := server.Send(&msg); err != nil {
				s.Log.Warn().Err(err).Msg("Send clock actual failed")
				return err
			}
		case <-ctx.Done():
			// Context canceled
			return nil
		}
	}
}

// Power is used to send power requests and receive power request & actual changes.
func (s *service) Power(server api.NetworkControlService_PowerServer) error {
	g, ctx := errgroup.WithContext(server.Context())

	// Incoming
	g.Go(func() error {
		for {
			msg, err := server.Recv()
			if util.IsStreamClosed(err) {
				return nil
			} else if err != nil {
				return err
			}
			s.Manager.SetPowerRequest(*msg)
		}
	})

	// Outgoing
	g.Go(func() error {
		ach, acancel := s.Manager.SubscribePowerActuals(true, chanTimeout)
		defer acancel()
		rch, rcancel := s.Manager.SubscribePowerRequests(true, chanTimeout)
		defer rcancel()
		for {
			select {
			case msg := <-ach:
				if err := server.Send(&msg); err != nil {
					return err
				}
			case msg := <-rch:
				if err := server.Send(&msg); err != nil {
					return err
				}
			case <-ctx.Done():
				// Context canceled
				return nil
			}
		}
	})
	return g.Wait()
}

// Locs is used to control locs and get changes in loc requests & actual state back.
// Note: Loc.actual on incoming objects is ignored.
func (s *service) Locs(server api.NetworkControlService_LocsServer) error {
	g, ctx := errgroup.WithContext(server.Context())

	// Incoming
	g.Go(func() error {
		for {
			msg, err := server.Recv()
			if util.IsStreamClosed(err) {
				return nil
			} else if err != nil {
				return err
			}
			s.Manager.SetLocRequest(*msg)
		}
	})

	// Outgoing
	g.Go(func() error {
		ach, acancel := s.Manager.SubscribeLocActuals(true, chanTimeout)
		defer acancel()
		rch, rcancel := s.Manager.SubscribeLocRequests(true, chanTimeout)
		defer rcancel()
		for {
			select {
			case msg := <-ach:
				if err := server.Send(&msg); err != nil {
					return err
				}
			case msg := <-rch:
				if err := server.Send(&msg); err != nil {
					return err
				}
			case <-ctx.Done():
				// Context canceled
				return nil
			}
		}
	})
	return g.Wait()
}

// Sensors is used to receive a stream of actual sensor states.
func (s *service) Sensors(req *api.Empty, server api.NetworkControlService_SensorsServer) error {
	g, ctx := errgroup.WithContext(server.Context())

	// Outgoing
	g.Go(func() error {
		ach, acancel := s.Manager.SubscribeSensorActuals(true, chanTimeout, "")
		defer acancel()
		for {
			select {
			case msg := <-ach:
				if err := server.Send(&msg); err != nil {
					return err
				}
			case <-ctx.Done():
				// Context canceled
				return nil
			}
		}
	})
	return g.Wait()
}

// Outputs is used to control outputs and get changes in output requests & actual state back.
// Note: Output.actual on incoming objects is ignored.
func (s *service) Outputs(server api.NetworkControlService_OutputsServer) error {
	g, ctx := errgroup.WithContext(server.Context())

	// Incoming
	g.Go(func() error {
		for {
			msg, err := server.Recv()
			if util.IsStreamClosed(err) {
				return nil
			} else if err != nil {
				return err
			}
			s.Manager.SetOutputRequest(*msg)
		}
	})

	// Outgoing
	g.Go(func() error {
		ach, acancel := s.Manager.SubscribeOutputActuals(true, chanTimeout, "")
		defer acancel()
		rch, rcancel := s.Manager.SubscribeOutputRequests(true, chanTimeout, "")
		defer rcancel()
		for {
			select {
			case msg := <-ach:
				if err := server.Send(&msg); err != nil {
					return err
				}
			case msg := <-rch:
				if err := server.Send(&msg); err != nil {
					return err
				}
			case <-ctx.Done():
				// Context canceled
				return nil
			}
		}
	})
	return g.Wait()
}

// Switches is used to control switches and get changes in switch requests & actual state back.
// Note: Switche.actual on incoming objects is ignored.
func (s *service) Switches(server api.NetworkControlService_SwitchesServer) error {
	g, ctx := errgroup.WithContext(server.Context())

	// Incoming
	g.Go(func() error {
		for {
			msg, err := server.Recv()
			if util.IsStreamClosed(err) {
				return nil
			} else if err != nil {
				return err
			}
			s.Manager.SetSwitchRequest(*msg)
		}
	})

	// Outgoing
	g.Go(func() error {
		ach, acancel := s.Manager.SubscribeSwitchActuals(true, chanTimeout, "")
		defer acancel()
		rch, rcancel := s.Manager.SubscribeSwitchRequests(true, chanTimeout, "")
		defer rcancel()
		for {
			select {
			case msg := <-ach:
				if err := server.Send(&msg); err != nil {
					return err
				}
			case msg := <-rch:
				if err := server.Send(&msg); err != nil {
					return err
				}
			case <-ctx.Done():
				// Context canceled
				return nil
			}
		}
	})
	return g.Wait()
}

// Clock is used to send clock requests and receive clock changes.
func (s *service) Clock(server api.NetworkControlService_ClockServer) error {
	g, ctx := errgroup.WithContext(server.Context())

	// Incoming
	g.Go(func() error {
		for {
			msg, err := server.Recv()
			if util.IsStreamClosed(err) {
				return nil
			} else if err != nil {
				return err
			}
			s.Manager.SetClockActual(*msg)
		}
	})

	// Outgoing
	g.Go(func() error {
		ach, acancel := s.Manager.SubscribeClockActuals(true, chanTimeout)
		defer acancel()
		for {
			select {
			case msg := <-ach:
				if err := server.Send(&msg); err != nil {
					return err
				}
			case <-ctx.Done():
				// Context canceled
				return nil
			}
		}
	})
	return g.Wait()
}
