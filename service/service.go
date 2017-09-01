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

	discoveryAPI "github.com/binkynet/BinkyNet/discovery"
	"github.com/rs/zerolog"

	"github.com/binkynet/NetManager/service/config"
	"github.com/binkynet/NetManager/service/discovery"
	"github.com/binkynet/NetManager/service/mqtt"
)

const (
	contentTypeJSON = "application/json"
)

type Service interface {
	// Run the manager until the given context is cancelled.
	Run(ctx context.Context) error
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
}

type Dependencies struct {
	Log               zerolog.Logger
	MqttService       mqtt.Service
	ConfigRegistry    config.Registry
	DiscoveryMessages <-chan discovery.RegisterWorkerMessage
}

type service struct {
	Config
	Dependencies
}

// NewService creates a Service instance and returns it.
func NewService(conf Config, deps Dependencies) (Service, error) {
	return &service{
		Config:       conf,
		Dependencies: deps,
	}, nil
}

// Run the manager until the given context is cancelled.
func (s *service) Run(ctx context.Context) error {
	for {
		select {
		case msg := <-s.DiscoveryMessages:
			// Process message
			go s.processDiscoveryMessage(msg.RemoteHost, msg.RegisterWorkerMessage)
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
	conf, err := s.ConfigRegistry.Get(msg.ID)
	if err != nil {
		s.Log.Error().Err(err).Str("id", msg.ID).Msg("Cannot open worker configuration")
		return
	}

	// Build environment message
	env := discoveryAPI.WorkerEnvironment{
		RequiredWorkerVersion: s.RequiredWorkerVersion,
	}
	env.Mqtt.Host = s.MQTTHost
	env.Mqtt.Port = s.MQTTPort
	env.Mqtt.UserName = s.MQTTUserName
	env.Mqtt.Password = s.MQTTPassword
	env.Mqtt.ControlTopic = conf.ControlTopic
	env.Mqtt.DataTopic = conf.DataTopic

	// Call environment endpoint
	scheme := "http"
	if msg.Secure {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s:%d/environment", scheme, remoteHost, msg.Port)
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
