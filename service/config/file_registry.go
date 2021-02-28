// Copyright 2021 Ewout Prangsma
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Author Ewout Prangsma
//

package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	model "github.com/binkynet/BinkyNet/apis/v1"
	yaml "gopkg.in/yaml.v2"
)

type registryEntry struct {
	model.LocalWorkerConfig
	modTime time.Time
}

// NewFileRegistry create a new Registry implementation backed by a filesystem.
func NewFileRegistry(ctx context.Context, folder string, reconfigureQueue chan string) (Registry, error) {
	r := &registry{
		folder:           folder,
		configs:          make(map[string]registryEntry),
		reconfigureQueue: reconfigureQueue,
	}
	go r.runMaintenance(ctx)
	return r, nil
}

type registry struct {
	mutex            sync.Mutex
	folder           string
	configs          map[string]registryEntry
	reconfigureQueue chan string
}

// Get returns the configuration for a worker with given ID.
func (r *registry) Get(id string) (model.LocalWorkerConfig, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if x, found := r.configs[id]; found {
		return x.LocalWorkerConfig, nil
	}

	// Read from disk
	conf, modTime, err := readWorkerConfiguration(r.folder, id)
	if err != nil {
		return model.LocalWorkerConfig{}, err
	}

	fmt.Printf("Read %+v\n", conf)

	// Save config
	r.configs[id] = registryEntry{
		LocalWorkerConfig: conf,
		modTime:           modTime,
	}

	return conf, nil
}

// runMaintenance keeps maintaining the registry until the given context is canceled.
func (r *registry) runMaintenance(ctx context.Context) {
	for {
		// Cleanup once
		r.removeChangedConfigs()

		select {
		case <-time.After(time.Second * 5):
			// Continue
		case <-ctx.Done():
			return
		}
	}
}

// removeChangedConfigs reads the modification time of the cache worker configs
// and removes those that have a different modification time or are no longer
// found.
func (r *registry) removeChangedConfigs() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for id, entry := range r.configs {
		_, modTime, err := readWorkerConfiguration(r.folder, id)
		if err != nil || modTime != entry.modTime {
			delete(r.configs, id)
			if r.reconfigureQueue != nil {
				r.reconfigureQueue <- id
			}
		}
	}
}

// readWorkerConfiguration tries to read the content of a configuration with given ID.
func readWorkerConfiguration(folder, id string) (model.LocalWorkerConfig, time.Time, error) {
	// Read file (yaml)
	filePath := filepath.Join(folder, id+".yaml")
	info, err := os.Stat(filePath)
	if err == nil {
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			return model.LocalWorkerConfig{}, time.Time{}, err
		}
		// Parse file
		var conf model.LocalWorkerConfig
		if err := yaml.Unmarshal(content, &conf); err != nil {
			return model.LocalWorkerConfig{}, time.Time{}, err
		}
		return conf, info.ModTime(), nil
	}

	// Read file (json)
	filePath = filepath.Join(folder, id+".json")
	info, err = os.Stat(filePath)
	if err == nil {
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			return model.LocalWorkerConfig{}, time.Time{}, err
		}

		// Parse file
		var conf model.LocalWorkerConfig
		if err := json.Unmarshal(content, &conf); err != nil {
			return model.LocalWorkerConfig{}, time.Time{}, err
		}
		return conf, info.ModTime(), nil
	}
	return model.LocalWorkerConfig{}, time.Time{}, err
}
