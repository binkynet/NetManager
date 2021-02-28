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
	api "github.com/binkynet/BinkyNet/apis/v1"
)

// GetConfig is used to fetch the configuration for a local worker.
func (s *service) GetConfig(req *api.LocalWorkerInfo, server api.LocalWorkerConfigService_GetConfigServer) error {
	id := req.GetId()
	log := s.Log.With().Str("id", id).Logger()
	if id == "" {
		return api.InvalidArgument("Id missing")
	}
	ctx := server.Context()
	log.Debug().Msg("GetConfig")

	// Subscribe to config changes
	configChanged := make(chan struct{})
	defer close(configChanged)
	onConfigChanged := func(idThatChanged string) {
		if id == idThatChanged {
			configChanged <- struct{}{}
		}
	}
	s.configChanges.Sub(onConfigChanged)
	defer s.configChanges.Leave(onConfigChanged)

	for {
		// Fetch config
		cfg, err := s.ConfigRegistry.Get(id)
		if err != nil {
			return api.NotFound("Failed to get configuration for '%s': %s", id, err.Error())
		}
		// Send config
		log.Debug().Msg("Sending config...")
		if err := server.Send(&cfg); err != nil {
			log.Debug().Err(err).Msg("Send(config) failed")
			return err
		}
		select {
		case <-ctx.Done():
			// Context canceled
			return nil
		case <-configChanged:
			// Resend config
		}
	}
}
