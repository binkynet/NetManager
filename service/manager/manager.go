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
	"fmt"
	"time"

	"github.com/mattn/go-pubsub"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/rs/zerolog"

	api "github.com/binkynet/BinkyNet/apis/v1"
)

// Manager is the abstraction of the core of the network manager.
type Manager interface {
	// Run the manager until the given context is cancelled.
	Run(ctx context.Context) error

	// GetLocalWorkerInfo fetches the last known info for a local worker with given ID.
	// Returns: info, remoteAddr, lastUpdatedAt, found
	GetLocalWorkerInfo(id string) (api.LocalWorkerInfo, string, time.Time, bool)
	// GetAllLocalWorkers fetches the last known info for all local workers.
	GetAllLocalWorkers() []api.LocalWorkerInfo
	// SubscribeLocalWorkerRequests is used to subscribe to requested changes of local workers.
	SubscribeLocalWorkerRequests(enabled bool, timeout time.Duration, filter ModuleFilter) (chan api.LocalWorker, context.CancelFunc)
	// SubscribeLocalWorkerActuals is used to subscribe to actual changes of local workers.
	SubscribeLocalWorkerActuals(enabled bool, timeout time.Duration, filter ModuleFilter) (chan api.LocalWorker, context.CancelFunc)
	// SetLocalWorkerRequest sets the requested state of a local worker
	SetLocalWorkerRequest(ctx context.Context, info api.LocalWorker) error
	// SetLocalWorkerActual sets the actual state of a local worker
	SetLocalWorkerActual(ctx context.Context, info api.LocalWorker, remoteAddr string) error
	// RequestResetLocalWorker requests the local worker with given ID to reset itself.
	RequestResetLocalWorker(ctx context.Context, id string)

	// Trigger a discovery and wait for the response.
	Discover(ctx context.Context, id string) (*api.DiscoverResult, error)
	// Trigger a discovery.
	SetDevicesDiscoveryRequest(ctx context.Context, req api.DeviceDiscovery)
	// SetDevicesDiscoveryActual is called by the local worker in response to discover requests.
	SetDevicesDiscoveryActual(ctx context.Context, req api.DeviceDiscovery) error
	// Subscribe to discovery actuals
	SubscribeDiscoverActuals(enabled bool, timeout time.Duration, id string) (chan api.DeviceDiscovery, context.CancelFunc)

	// Set the requested power state
	SetPowerRequest(x api.PowerState)
	// Set the actual power state
	SetPowerActual(x api.PowerState)
	// Subscribe to power actuals
	SubscribePowerActuals(enabled bool, timeout time.Duration) (chan api.Power, context.CancelFunc)

	// Set the requested loc state
	SetLocRequest(x api.Loc)
	// Set the actual loc state
	SetLocActual(x api.Loc)
	// Subscribe to loc actuals
	SubscribeLocActuals(enabled bool, timeout time.Duration) (chan api.Loc, context.CancelFunc)

	// Set the requested output state
	SetOutputRequest(x api.Output)
	// Set the actual output state
	SetOutputActual(x api.Output)
	// Subscribe to output actuals
	SubscribeOutputActuals(enabled bool, timeout time.Duration, filter ModuleFilter) (chan api.Output, context.CancelFunc)

	// Set the actual sensor state
	SetSensorActual(x api.Sensor)
	// Subscribe to sensor actuals
	SubscribeSensorActuals(enabled bool, timeout time.Duration, filter ModuleFilter) (chan api.Sensor, context.CancelFunc)

	// Set the requested switch state
	SetSwitchRequest(x api.Switch)
	// Set the actual switch state
	SetSwitchActual(x api.Switch)
	// Subscribe to switch actuals
	SubscribeSwitchActuals(enabled bool, timeout time.Duration, filter ModuleFilter) (chan api.Switch, context.CancelFunc)

	// Set the actual clock state
	SetClockActual(x api.Clock)
	// Subscribe to clock actuals
	SubscribeClockActuals(enabled bool, timeout time.Duration) (chan api.Clock, context.CancelFunc)
}

// Dependencies of the manager.
type Dependencies struct {
	Log zerolog.Logger

	// Reconfiguration queue (chan localWorkerID).
	// The manager must listen to entries in this queue and reconfigure
	// when it receives a local worker ID.
	ReconfigureQueue <-chan string
}

// New creates a new Manager.
func New(deps Dependencies) (Manager, error) {
	// Prepare MQTT server
	options := &mqtt.Options{
		InlineClient: true,
	}
	mqttServer := mqtt.New(options)
	// For security reasons, the default implementation disallows all connections.
	// If you want to allow all connections, you must specifically allow it.
	if err := mqttServer.AddHook(new(auth.AllowHook), nil); err != nil {
		return nil, fmt.Errorf("MQTT Hook configuration failed: %w", err)
	}

	return &manager{
		Dependencies:    deps,
		mqttServer:      mqttServer,
		configChanges:   pubsub.New(),
		discoverPool:    newDiscoverPool(deps.Log),
		powerPool:       newPowerPool(deps.Log),
		locPool:         newLocPool(deps.Log),
		outputPool:      newOutputPool(deps.Log),
		sensorPool:      newSensorPool(deps.Log),
		switchPool:      newSwitchPool(deps.Log),
		clockPool:       newClockPool(deps.Log),
		localWorkerPool: newLocalWorkerPool(deps.Log),
	}, nil

}

// manager implements the core of the network manager
type manager struct {
	Dependencies

	mqttServer      *mqtt.Server
	configChanges   *pubsub.PubSub
	discoverPool    *discoverPool
	powerPool       *powerPool
	locPool         *locPool
	outputPool      *outputPool
	sensorPool      *sensorPool
	switchPool      *switchPool
	clockPool       *clockPool
	localWorkerPool *localWorkerPool
}

// Run the manager until the given context is cancelled.
func (m *manager) Run(ctx context.Context) error {
	log := m.Log.With().Str("component", "service").Logger()
	defer func() {
		log.Debug().Msg("Run finished")
	}()

	// Prepare listener
	tcp := listeners.NewTCP(listeners.Config{
		ID:      "t1",
		Address: ":1883",
	})
	if err := m.mqttServer.AddListener(tcp); err != nil {
		return fmt.Errorf("MQTT Listener configuration failed: %w", err)
	}
	go func() {
		if err := m.mqttServer.Serve(); err != nil {
			log.Error().Err(err).Msg("MQTT Serve failed")
		}
	}()

	for {
		select {
		case id := <-m.ReconfigureQueue:
			// Reconfigure worker with id
			log.Info().Str("id", id).Msg("Reconfiguration detected")
			m.configChanges.Pub(id)
		case <-ctx.Done():
			// Context cancelled
			m.mqttServer.Close()
			tcp.Close(func(id string) {
				// Do nothing
			})
			return nil
		}
	}
}

// GetLocalWorkerInfo fetches the last known info for a local worker with given ID.
// Returns: LWinfo, LastUpdatedAt, found
func (m *manager) GetLocalWorkerInfo(id string) (api.LocalWorkerInfo, string, time.Time, bool) {
	return m.localWorkerPool.GetInfo(id)
}

// GetAllLocalWorkers fetches the last known info for all local workers.
func (m *manager) GetAllLocalWorkers() []api.LocalWorkerInfo {
	return m.localWorkerPool.GetAll()
}

// SubscribeLocalWorkerRequests is used to subscribe to requested changes of local workers.
func (m *manager) SubscribeLocalWorkerRequests(enabled bool, timeout time.Duration, filter ModuleFilter) (chan api.LocalWorker, context.CancelFunc) {
	return m.localWorkerPool.SubRequests(enabled, timeout, filter)
}

// SubscribeLocalWorkerActuals is used to subscribe to actual changes of local workers.
func (m *manager) SubscribeLocalWorkerActuals(enabled bool, timeout time.Duration, filter ModuleFilter) (chan api.LocalWorker, context.CancelFunc) {
	return m.localWorkerPool.SubActuals(enabled, timeout, filter)
}

// SetLocalWorkerRequest sets the requested state of a local worker
func (m *manager) SetLocalWorkerRequest(ctx context.Context, lw api.LocalWorker) error {
	return m.localWorkerPool.SetRequest(ctx, lw)
}

// SetLocalWorkerActual sets the actual state of a local worker
func (m *manager) SetLocalWorkerActual(ctx context.Context, lw api.LocalWorker, remoteAddr string) error {
	return m.localWorkerPool.SetActual(ctx, lw, remoteAddr)
}

// RequestResetLocalWorker requests the local worker with given ID to reset itself.
func (m *manager) RequestResetLocalWorker(ctx context.Context, id string) {
	m.localWorkerPool.RequestReset(ctx, id)
}

// Trigger a discovery and wait for the response.
func (m *manager) Discover(ctx context.Context, id string) (*api.DiscoverResult, error) {
	m.Log.Debug().Msg("manager.Discover")
	return m.discoverPool.Trigger(ctx, id)
}

// Trigger a discovery.
func (m *manager) SetDevicesDiscoveryRequest(ctx context.Context, req api.DeviceDiscovery) {
	m.discoverPool.SetDiscoverRequest(req)
	log := m.Log
	go func() {
		lwInfo, _, _, _ := m.localWorkerPool.GetInfo(req.GetId())
		if lwInfo.GetSupportsSetDeviceDiscoveryRequest() {
			if client, err := m.localWorkerPool.GetLocalWorkerServiceClient(lwInfo.GetId()); err != nil {
				log.Error().Err(err).
					Str("id", lwInfo.GetId()).
					Msg("Failed to get local worker client")
			} else {
				if _, err := client.SetDeviceDiscoveryRequest(context.Background(), &req); err != nil {
					log.Error().Err(err).
						Str("id", lwInfo.GetId()).
						Msg("Failed to send device discovery request to local worker")
				}
			}
		}
	}()

}

// SetDevicesDiscoveryActual is called by the local worker in response to discover requests.
func (m *manager) SetDevicesDiscoveryActual(ctx context.Context, req api.DeviceDiscovery) error {
	return m.discoverPool.SetDiscoverResult(req)
}

// Subscribe to discovery requests
func (m *manager) SubscribeDiscoverRequests(enabled bool, timeout time.Duration, id string) (chan api.DeviceDiscovery, context.CancelFunc) {
	return m.discoverPool.SubRequests(enabled, timeout, id)
}

// Subscribe to discovery actuals
func (m *manager) SubscribeDiscoverActuals(enabled bool, timeout time.Duration, id string) (chan api.DeviceDiscovery, context.CancelFunc) {
	return m.discoverPool.SubActuals(enabled, timeout, id)
}

// Set the requested power state
func (m *manager) SetPowerRequest(x api.PowerState) {
	m.powerPool.SetRequest(x)
	log := m.Log
	go func() {
		for _, lwInfo := range m.localWorkerPool.GetAll() {
			if lwInfo.GetSupportsSetPowerRequest() {
				if client, err := m.localWorkerPool.GetLocalWorkerServiceClient(lwInfo.GetId()); err != nil {
					log.Error().Err(err).
						Str("id", lwInfo.GetId()).
						Msg("Failed to get local worker client")
				} else {
					if _, err := client.SetPowerRequest(context.Background(), &x); err != nil {
						log.Error().Err(err).
							Str("id", lwInfo.GetId()).
							Msg("Failed to send power request to local worker")
					}
				}
			}
		}
	}()
}

// Set the actual power state
func (m *manager) SetPowerActual(x api.PowerState) {
	m.powerPool.SetActual(x)
}

// Subscribe to power actuals
func (m *manager) SubscribePowerActuals(enabled bool, timeout time.Duration) (chan api.Power, context.CancelFunc) {
	return m.powerPool.SubActual(enabled, timeout)
}

// Set the requested loc state
func (m *manager) SetLocRequest(x api.Loc) {
	m.locPool.SetRequest(x)
	log := m.Log
	go func() {
		for _, lwInfo := range m.localWorkerPool.GetAll() {
			if lwInfo.GetSupportsSetLocRequest() {
				if client, err := m.localWorkerPool.GetLocalWorkerServiceClient(lwInfo.GetId()); err != nil {
					log.Error().Err(err).
						Str("id", lwInfo.GetId()).
						Msg("Failed to get local worker client")
				} else {
					if _, err := client.SetLocRequest(context.Background(), &x); err != nil {
						log.Error().Err(err).
							Str("id", lwInfo.GetId()).
							Msg("Failed to send loc request to local worker")
					}
				}
			}
		}
	}()
}

// Set the actual loc state
func (m *manager) SetLocActual(x api.Loc) {
	m.locPool.SetActual(x)
}

// Subscribe to loc actuals
func (m *manager) SubscribeLocActuals(enabled bool, timeout time.Duration) (chan api.Loc, context.CancelFunc) {
	return m.locPool.SubActual(enabled, timeout)
}

// Set the requested output state
func (m *manager) SetOutputRequest(x api.Output) {
	m.outputPool.SetRequest(x)
	log := m.Log
	go func() {
		moduleID, _, _ := api.SplitAddress(x.Address)
		if moduleID == api.GlobalModuleID {
			// Send to all
			for _, lwInfo := range m.localWorkerPool.GetAll() {
				if lwInfo.GetSupportsSetOutputRequest() {
					if client, err := m.localWorkerPool.GetLocalWorkerServiceClient(lwInfo.GetId()); err != nil {
						log.Error().Err(err).
							Str("id", lwInfo.GetId()).
							Msg("Failed to get local worker client")
					} else {
						if _, err := client.SetOutputRequest(context.Background(), &x); err != nil {
							log.Error().Err(err).
								Str("id", lwInfo.GetId()).
								Msg("Failed to send output request to local worker")
						}
					}
				}
			}
		} else {
			lwInfo, _, _, _ := m.localWorkerPool.GetInfo(moduleID)
			if lwInfo.GetSupportsSetOutputRequest() {
				if client, err := m.localWorkerPool.GetLocalWorkerServiceClient(lwInfo.GetId()); err != nil {
					log.Error().Err(err).
						Str("id", lwInfo.GetId()).
						Msg("Failed to get local worker client")
				} else {
					if _, err := client.SetOutputRequest(context.Background(), &x); err != nil {
						log.Error().Err(err).
							Str("id", lwInfo.GetId()).
							Msg("Failed to send output request to local worker")
					}
				}
			}
		}
	}()
}

// Set the actual output state
func (m *manager) SetOutputActual(x api.Output) {
	m.outputPool.SetActual(x)
}

// Subscribe to output actuals
func (m *manager) SubscribeOutputActuals(enabled bool, timeout time.Duration, filter ModuleFilter) (chan api.Output, context.CancelFunc) {
	return m.outputPool.SubActual(enabled, timeout, filter)
}

// Set the actual sensor state
func (m *manager) SetSensorActual(x api.Sensor) {
	m.sensorPool.SetActual(x)
}

// Subscribe to sensor actuals
func (m *manager) SubscribeSensorActuals(enabled bool, timeout time.Duration, filter ModuleFilter) (chan api.Sensor, context.CancelFunc) {
	return m.sensorPool.SubActual(enabled, timeout, filter)
}

// Set the requested switch state
func (m *manager) SetSwitchRequest(x api.Switch) {
	m.switchPool.SetRequest(x)
	log := m.Log
	go func() {
		moduleID, _, _ := api.SplitAddress(x.Address)
		if moduleID == api.GlobalModuleID {
			// Send to all
			for _, lwInfo := range m.localWorkerPool.GetAll() {
				if lwInfo.GetSupportsSetSwitchRequest() {
					if client, err := m.localWorkerPool.GetLocalWorkerServiceClient(lwInfo.GetId()); err != nil {
						log.Error().Err(err).
							Str("id", lwInfo.GetId()).
							Msg("Failed to get local worker client")
					} else {
						if _, err := client.SetSwitchRequest(context.Background(), &x); err != nil {
							log.Error().Err(err).
								Str("id", lwInfo.GetId()).
								Msg("Failed to send switch request to local worker")
						}
					}
				}
			}
		} else {
			lwInfo, _, _, _ := m.localWorkerPool.GetInfo(moduleID)
			if lwInfo.GetSupportsSetSwitchRequest() {
				if client, err := m.localWorkerPool.GetLocalWorkerServiceClient(lwInfo.GetId()); err != nil {
					log.Error().Err(err).
						Str("id", lwInfo.GetId()).
						Msg("Failed to get local worker client")
				} else {
					if _, err := client.SetSwitchRequest(context.Background(), &x); err != nil {
						log.Error().Err(err).
							Str("id", lwInfo.GetId()).
							Msg("Failed to send switch request to local worker")
					}
				}
			}
		}
	}()
}

// Set the actual switch state
func (m *manager) SetSwitchActual(x api.Switch) {
	m.switchPool.SetActual(x)
}

// Subscribe to switch actuals
func (m *manager) SubscribeSwitchActuals(enabled bool, timeout time.Duration, filter ModuleFilter) (chan api.Switch, context.CancelFunc) {
	return m.switchPool.SubActual(enabled, timeout, filter)
}

// Set the actual clock state
func (m *manager) SetClockActual(x api.Clock) {
	m.clockPool.SetActual(x)
}

// Subscribe to clock actuals
func (m *manager) SubscribeClockActuals(enabled bool, timeout time.Duration) (chan api.Clock, context.CancelFunc) {
	return m.clockPool.SubActual(enabled, timeout)
}
