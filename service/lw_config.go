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
	"time"

	api "github.com/binkynet/BinkyNet/apis/v1"
	"github.com/binkynet/NetManager/service/manager"
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
	rch, cancel := s.Manager.SubscribeLocalWorkerRequests(true, manager.ModuleFilter(req.GetId()))
	defer cancel()

	for {
		select {
		case lw := <-rch:
			if lw.GetId() == id {
				if cfg := lw.GetRequest(); cfg != nil {
					log.Debug().Msg("Sending config...")
					cfg.Unixtime = time.Now().Unix()
					if err := server.Send(cfg); err != nil {
						log.Debug().Err(err).Msg("Send(config) failed")
						return err
					}
				}
			}
		case <-ctx.Done():
			// Context canceled
			return nil
		}
	}
}
