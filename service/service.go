//    Copyright 2017 Ewout Prangsma
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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	discoveryAPI "github.com/binkynet/BinkyNet/discovery"
	"github.com/binkynet/BinkyNet/model"
	"github.com/rs/zerolog"

	"github.com/binkynet/NetManager/client"
	"github.com/binkynet/NetManager/service/config"
	"github.com/binkynet/NetManager/service/discovery"
)

const (
	contentTypeJSON = "application/json"
)

// Service is the API exposed by this service.
type Service interface {
	// Run the manager until the given context is cancelled.
	Run(ctx context.Context) error
	client.API
}

type Config struct {
	// LocalWorker version (semver) that is expected.
	// If the actual version is different, the LocalWorker must update
	// itself.
	RequiredWorkerVersion string
	// MQTT server host
	MQTTHost string
	// MQTT server port
	MQTTPort int
	// MQTT user name for authentication
	MQTTUserName string
	// MQTT password for authentication
	MQTTPassword string
	// Prefix for topics in MQTT
	MQTTTopicPrefix string
	// Endpoint of NetManager.
	MyEndpoint string
}

type Dependencies struct {
	Log zerolog.Logger

	ConfigRegistry    config.Registry
	DiscoveryMessages <-chan discovery.RegisterWorkerMessage
	ReconfigureQueue  <-chan string
}

type service struct {
	Config
	Dependencies

	mutex   sync.RWMutex
	workers map[string]workerRegistration
}

// NewService creates a Service instance and returns it.
func NewService(conf Config, deps Dependencies) (Service, error) {
	return &service{
		Config:       conf,
		Dependencies: deps,
		workers:      make(map[string]workerRegistration),
	}, nil
}

// Run the manager until the given context is cancelled.
func (s *service) Run(ctx context.Context) error {
	for {
		select {
		case msg := <-s.DiscoveryMessages:
			// Process message
			go s.processDiscoveryMessage(msg.RemoteHost, msg.RegisterWorkerMessage)
		case id := <-s.ReconfigureQueue:
			// Reconfigure worker
			go s.ReconfigureWorker(ctx, id)
		case <-ctx.Done():
			// Context cancalled
			return nil
		}
	}
}

// processDiscoveryMessage process the given message.
// It calls the environment route of the local worker that has registered.
func (s *service) processDiscoveryMessage(remoteHost string, msg discoveryAPI.RegisterWorkerMessage) {
	if msg.ID == "" {
		s.Log.Error().Msg("Received RegisterWorkerMessage with empty ID")
		return
	}
	// Find config
	_, err := s.ConfigRegistry.Get(msg.ID)
	if err != nil {
		s.Log.Error().Err(err).Str("id", msg.ID).Msg("Cannot open worker configuration")
		return
	}

	// Store registration
	reg := workerRegistration{
		RegisterWorkerMessage: discovery.RegisterWorkerMessage{
			RegisterWorkerMessage: msg,
			RemoteHost:            remoteHost,
		},
	}
	s.mutex.Lock()
	s.workers[msg.ID] = reg
	s.mutex.Unlock()

	// Build environment message
	env := discoveryAPI.WorkerEnvironment{
		RequiredWorkerVersion: s.RequiredWorkerVersion,
	}
	env.Mqtt.Host = s.MQTTHost
	env.Mqtt.Port = s.MQTTPort
	env.Mqtt.UserName = s.MQTTUserName
	env.Mqtt.Password = s.MQTTPassword
	env.Mqtt.TopicPrefix = s.MQTTTopicPrefix
	env.Manager.Endpoint = s.MyEndpoint

	// Call environment endpoint
	url := reg.Endpoint("/environment")
	encodedEnv, err := json.Marshal(env)
	if err != nil {
		s.Log.Error().Err(err).Str("id", msg.ID).Msg("Failed to encode environment information")
		return
	}
	if resp, err := http.Post(url, contentTypeJSON, bytes.NewReader(encodedEnv)); err != nil {
		s.Log.Error().Err(err).Str("id", msg.ID).Msg("Failed call environment endpoint")
	} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.Log.Error().Str("id", msg.ID).Int("status", resp.StatusCode).Msg("Unexpected status code from environment call")
	} else {
		s.Log.Debug().Str("id", msg.ID).Msg("Call to environment endpoint succeeded")
	}
}

// ReconfigureWorker calls the worker to perform a complete reload.
func (s *service) ReconfigureWorker(ctx context.Context, workerID string) error {
	s.mutex.Lock()
	reg, found := s.workers[workerID]
	s.mutex.Unlock()

	if !found {
		return maskAny(fmt.Errorf("No such worker: %s", workerID))
	}

	url := reg.Endpoint("/environment")
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return maskAny(err)
	}
	lctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	req = req.WithContext(lctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return maskAny(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.Log.Error().Str("id", workerID).Int("status", resp.StatusCode).Msg("Unexpected status code from DELETE environment call")
		return maskAny(fmt.Errorf("Unexpected status code %d from DELETE environment call", resp.StatusCode))
	}
	s.Log.Debug().Str("id", workerID).Msg("Call to environment endpoint succeeded")
	return nil
}

// ShutdownWorker calls the worker to perform a complete reload.
func (s *service) ShutdownWorker(ctx context.Context, workerID string) error {
	s.mutex.Lock()
	reg, found := s.workers[workerID]
	s.mutex.Unlock()

	if !found {
		return maskAny(fmt.Errorf("No such worker: %s", workerID))
	}

	url := reg.Endpoint("/shutdown")
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return maskAny(err)
	}
	lctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	req = req.WithContext(lctx)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return maskAny(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.Log.Error().Str("id", workerID).Int("status", resp.StatusCode).Msg("Unexpected status code from POST shutdown call")
		return maskAny(fmt.Errorf("Unexpected status code %d from POST shutdown call", resp.StatusCode))
	}
	s.Log.Debug().Str("id", workerID).Msg("Call to environment endpoint succeeded")
	return nil
}

// Get the configuration for a specific local worker
func (s *service) GetWorkerConfig(ctx context.Context, workerID string) (model.LocalWorkerConfig, error) {
	conf, err := s.ConfigRegistry.Get(workerID)
	if err != nil {
		s.Log.Error().Err(err).Str("id", workerID).Msg("Cannot open worker configuration")
		return model.LocalWorkerConfig{}, maskAny(err)
	}
	return conf.LocalWorkerConfig, nil
}

// GetWorkers returns a list of registered workers
func (s *service) GetWorkers(ctx context.Context) ([]client.WorkerInfo, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make([]client.WorkerInfo, 0, len(s.workers))
	for id, reg := range s.workers {
		result = append(result, client.WorkerInfo{
			ID:       id,
			Endpoint: reg.Endpoint("/"),
		})
	}
	return result, nil
}
