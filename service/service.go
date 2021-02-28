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
	model "github.com/binkynet/BinkyNet/apis/v1"
	"github.com/mattn/go-pubsub"
	"github.com/rs/zerolog"

	"github.com/binkynet/NetManager/service/config"
	"github.com/binkynet/NetManager/service/manager"
)

const (
	contentTypeJSON = "application/json"
)

// Service is the API exposed by this service.
type Service interface {
	model.LocalWorkerConfigServiceServer
	model.LocalWorkerControlServiceServer
	model.NetworkControlServiceServer
}

type Config struct {
	// LocalWorker version (semver) that is expected.
	// If the actual version is different, the LocalWorker must update
	// itself.
	RequiredWorkerVersion string
}

type Dependencies struct {
	Log zerolog.Logger

	Manager        manager.Manager
	ConfigRegistry config.Registry
}

type service struct {
	Config
	Dependencies

	configChanges *pubsub.PubSub
}

// NewService creates a Service instance and returns it.
func NewService(conf Config, deps Dependencies) (Service, error) {
	return &service{
		Config:        conf,
		Dependencies:  deps,
		configChanges: pubsub.New(),
	}, nil
}
