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
	"time"

	api "github.com/binkynet/BinkyNet/apis/v1"
	"github.com/binkynet/NetManager/service/config"
	"github.com/mattn/go-pubsub"
	"github.com/rs/zerolog"
)

// Manager is the abstraction of the core of the network manager.
type Manager interface {
	// Run the manager until the given context is cancelled.
	Run(ctx context.Context) error

	// GetLocalWorkerInfo fetches the last known info for a local worker with given ID.
	// Returns: info, lastUpdatedAt, found
	GetLocalWorkerInfo(id string) (api.LocalWorkerInfo, time.Time, bool)
	// GetAllLocalWorkers fetches the last known info for all local workers.
	GetAllLocalWorkers() []api.LocalWorkerInfo
	// SubscribeLocalWorkerUpdates is used to subscribe to updates of all local workers.
	SubscribeLocalWorkerUpdates() (chan api.LocalWorkerInfo, context.CancelFunc)
	// SetLocalWorkerUpdate informs the pool and its listeners of a local worker update.
	SetLocalWorkerUpdate(ctx context.Context, info api.LocalWorkerInfo)

	// Trigger a discovery and wait for the response.
	Discover(ctx context.Context, id string) (*api.DiscoverResult, error)
	// Subscribe to power requests
	SubscribeDiscoverRequests(id string) (chan api.DiscoverRequest, context.CancelFunc)
	// SetDiscoverResult is called by the local worker in response to discover requests.
	SetDiscoverResult(ctx context.Context, req api.DiscoverResult) error

	// Set the requested power state
	SetPowerRequest(x api.PowerState)
	// Set the actual power state
	SetPowerActual(x api.PowerState)
	// Subscribe to power requests
	SubscribePowerRequests(enabled bool) (chan api.Power, context.CancelFunc)
	// Subscribe to power actuals
	SubscribePowerActuals(enabled bool) (chan api.Power, context.CancelFunc)

	// Set the requested loc state
	SetLocRequest(x api.Loc)
	// Set the actual loc state
	SetLocActual(x api.Loc)
	// Subscribe to loc requests
	SubscribeLocRequests(enabled bool) (chan api.Loc, context.CancelFunc)
	// Subscribe to loc actuals
	SubscribeLocActuals(enabled bool) (chan api.Loc, context.CancelFunc)

	// Set the requested output state
	SetOutputRequest(x api.Output)
	// Set the actual output state
	SetOutputActual(x api.Output)
	// Subscribe to output requests
	SubscribeOutputRequests(enabled bool) (chan api.Output, context.CancelFunc)
	// Subscribe to output actuals
	SubscribeOutputActuals(enabled bool) (chan api.Output, context.CancelFunc)

	// Set the actual sensor state
	SetSensorActual(x api.Sensor)
	// Subscribe to sensor actuals
	SubscribeSensorActuals(enabled bool) (chan api.Sensor, context.CancelFunc)

	// Set the requested switch state
	SetSwitchRequest(x api.Switch)
	// Set the actual switch state
	SetSwitchActual(x api.Switch)
	// Subscribe to switch requests
	SubscribeSwitchRequests(enabled bool) (chan api.Switch, context.CancelFunc)
	// Subscribe to switch actuals
	SubscribeSwitchActuals(enabled bool) (chan api.Switch, context.CancelFunc)

	// Set the actual clock state
	SetClockActual(x api.Clock)
	// Subscribe to clock actuals
	SubscribeClockActuals(enabled bool) (chan api.Clock, context.CancelFunc)
}

// Dependencies of the manager.
type Dependencies struct {
	Log zerolog.Logger

	// Local worker registry
	ConfigRegistry config.Registry
	// Reconfiguration queue (chan localWorkerID).
	// The manager must listen to entries in this queue and reconfigure
	// when it receives a local worker ID.
	ReconfigureQueue <-chan string
}

// New creates a new Manager.
func New(deps Dependencies) (Manager, error) {
	return &manager{
		Dependencies:    deps,
		configChanges:   pubsub.New(),
		discoverPool:    newDiscoverPool(deps.Log),
		powerPool:       newPowerPool(),
		locPool:         newLocPool(),
		outputPool:      newOutputPool(),
		sensorPool:      newSensorPool(),
		switchPool:      newSwitchPool(),
		clockPool:       newClockPool(),
		localWorkerPool: newLocalWorkerPool(deps.Log),
	}, nil

}

// manager implements the core of the network manager
type manager struct {
	Dependencies

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
	for {
		select {
		case id := <-m.ReconfigureQueue:
			// Reconfigure worker with id
			log.Info().Str("id", id).Msg("Reconfiguration detected")
			m.configChanges.Pub(id)
		case <-ctx.Done():
			// Context cancalled
			return nil
		}
	}
}

// GetLocalWorkerInfo fetches the last known info for a local worker with given ID.
// Returns: LWinfo, LastUpdatedAt, found
func (m *manager) GetLocalWorkerInfo(id string) (api.LocalWorkerInfo, time.Time, bool) {
	return m.localWorkerPool.GetInfo(id)
}

// GetAllLocalWorkers fetches the last known info for all local workers.
func (m *manager) GetAllLocalWorkers() []api.LocalWorkerInfo {
	return m.localWorkerPool.GetAll()
}

// SubscribeLocalWorkerUpdates is used to subscribe to updates of all local workers.
func (m *manager) SubscribeLocalWorkerUpdates() (chan api.LocalWorkerInfo, context.CancelFunc) {
	return m.localWorkerPool.SubUpdates()
}

// SetLocalWorkerUpdate informs the pool and its listeners of a local worker update.
func (m *manager) SetLocalWorkerUpdate(ctx context.Context, info api.LocalWorkerInfo) {
	m.localWorkerPool.SetUpdate(ctx, info)
}

// Trigger a discovery and wait for the response.
func (m *manager) Discover(ctx context.Context, id string) (*api.DiscoverResult, error) {
	m.Log.Debug().Msg("manager.Discover")
	return m.discoverPool.Trigger(ctx, id)
}

// Subscribe to power requests
func (m *manager) SubscribeDiscoverRequests(id string) (chan api.DiscoverRequest, context.CancelFunc) {
	return m.discoverPool.SubRequest(id)
}

// SetDiscoverResult is called by the local worker in response to discover requests.
func (m *manager) SetDiscoverResult(ctx context.Context, req api.DiscoverResult) error {
	return m.discoverPool.SetDiscoverResult(req)
}

// Set the requested power state
func (m *manager) SetPowerRequest(x api.PowerState) {
	m.powerPool.SetRequest(x)
}

// Set the actual power state
func (m *manager) SetPowerActual(x api.PowerState) {
	m.powerPool.SetActual(x)
}

// Subscribe to power requests
func (m *manager) SubscribePowerRequests(enabled bool) (chan api.Power, context.CancelFunc) {
	return m.powerPool.SubRequest(enabled)
}

// Subscribe to power actuals
func (m *manager) SubscribePowerActuals(enabled bool) (chan api.Power, context.CancelFunc) {
	return m.powerPool.SubActual(enabled)
}

// Set the requested loc state
func (m *manager) SetLocRequest(x api.Loc) {
	m.locPool.SetRequest(x)
}

// Set the actual loc state
func (m *manager) SetLocActual(x api.Loc) {
	m.locPool.SetActual(x)
}

// Subscribe to loc requests
func (m *manager) SubscribeLocRequests(enabled bool) (chan api.Loc, context.CancelFunc) {
	return m.locPool.SubRequest(enabled)
}

// Subscribe to loc actuals
func (m *manager) SubscribeLocActuals(enabled bool) (chan api.Loc, context.CancelFunc) {
	return m.locPool.SubActual(enabled)
}

// Set the requested output state
func (m *manager) SetOutputRequest(x api.Output) {
	m.outputPool.SetRequest(x)
}

// Set the actual output state
func (m *manager) SetOutputActual(x api.Output) {
	m.outputPool.SetActual(x)
}

// Subscribe to output requests
func (m *manager) SubscribeOutputRequests(enabled bool) (chan api.Output, context.CancelFunc) {
	return m.outputPool.SubRequest(enabled)
}

// Subscribe to output actuals
func (m *manager) SubscribeOutputActuals(enabled bool) (chan api.Output, context.CancelFunc) {
	return m.outputPool.SubActual(enabled)
}

// Set the actual sensor state
func (m *manager) SetSensorActual(x api.Sensor) {
	m.sensorPool.SetActual(x)
}

// Subscribe to sensor actuals
func (m *manager) SubscribeSensorActuals(enabled bool) (chan api.Sensor, context.CancelFunc) {
	return m.sensorPool.SubActual(enabled)
}

// Set the requested switch state
func (m *manager) SetSwitchRequest(x api.Switch) {
	m.switchPool.SetRequest(x)
}

// Set the actual switch state
func (m *manager) SetSwitchActual(x api.Switch) {
	m.switchPool.SetActual(x)
}

// Subscribe to switch requests
func (m *manager) SubscribeSwitchRequests(enabled bool) (chan api.Switch, context.CancelFunc) {
	return m.switchPool.SubRequest(enabled)
}

// Subscribe to switch actuals
func (m *manager) SubscribeSwitchActuals(enabled bool) (chan api.Switch, context.CancelFunc) {
	return m.switchPool.SubActual(enabled)
}

// Set the actual clock state
func (m *manager) SetClockActual(x api.Clock) {
	m.clockPool.SetActual(x)
}

// Subscribe to clock actuals
func (m *manager) SubscribeClockActuals(enabled bool) (chan api.Clock, context.CancelFunc) {
	return m.clockPool.SubActual(enabled)
}
