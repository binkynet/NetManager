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
	"github.com/binkynet/BinkyNet/apis/util"
	api "github.com/binkynet/BinkyNet/apis/v1"
	"golang.org/x/sync/errgroup"
)

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
			s.powerPool.SetRequest(*msg)
		}
	})

	// Outgoing
	g.Go(func() error {
		ach, acancel := s.powerPool.SubActual()
		defer acancel()
		rch, rcancel := s.powerPool.SubRequest()
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
			s.locPool.SetRequest(*msg)
		}
	})

	// Outgoing
	g.Go(func() error {
		ach, acancel := s.locPool.SubActual()
		defer acancel()
		rch, rcancel := s.locPool.SubRequest()
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
		ach, acancel := s.sensorPool.SubActual()
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
			s.outputPool.SetRequest(*msg)
		}
	})

	// Outgoing
	g.Go(func() error {
		ach, acancel := s.outputPool.SubActual()
		defer acancel()
		rch, rcancel := s.outputPool.SubRequest()
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
			s.switchPool.SetRequest(*msg)
		}
	})

	// Outgoing
	g.Go(func() error {
		ach, acancel := s.switchPool.SubActual()
		defer acancel()
		rch, rcancel := s.switchPool.SubRequest()
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
			s.clockPool.SetActual(*msg)
		}
	})

	// Outgoing
	g.Go(func() error {
		ach, acancel := s.clockPool.SubActual()
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
