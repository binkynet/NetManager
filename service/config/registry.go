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

type Registry interface {
	// Get returns the configuration for a worker with given ID.
	Get(id string) (WorkerConfiguration, error)
}

// WorkerConfiguration file format for local worker configuration file.
type WorkerConfiguration struct {
	model.LocalWorkerConfig
}

type registryEntry struct {
	WorkerConfiguration
	modTime time.Time
}

// NewRegistry create a new Registry implementation.
func NewRegistry(ctx context.Context, folder string, reconfigureQueue chan string) (Registry, error) {
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
func (r *registry) Get(id string) (WorkerConfiguration, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if x, found := r.configs[id]; found {
		return x.WorkerConfiguration, nil
	}

	// Read from disk
	conf, modTime, err := readWorkerConfiguration(r.folder, id)
	if err != nil {
		return WorkerConfiguration{}, err
	}

	fmt.Printf("Read %+v\n", conf)

	// Save config
	r.configs[id] = registryEntry{
		WorkerConfiguration: conf,
		modTime:             modTime,
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
func readWorkerConfiguration(folder, id string) (WorkerConfiguration, time.Time, error) {
	// Read file (yaml)
	filePath := filepath.Join(folder, id+".yaml")
	info, err := os.Stat(filePath)
	if err == nil {
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			return WorkerConfiguration{}, time.Time{}, err
		}
		// Parse file
		var conf WorkerConfiguration
		if err := yaml.Unmarshal(content, &conf.LocalWorkerConfig); err != nil {
			return WorkerConfiguration{}, time.Time{}, err
		}
		return conf, info.ModTime(), nil
	}

	// Read file (json)
	filePath = filepath.Join(folder, id+".json")
	info, err = os.Stat(filePath)
	if err == nil {
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			return WorkerConfiguration{}, time.Time{}, err
		}

		// Parse file
		var conf WorkerConfiguration
		if err := json.Unmarshal(content, &conf.LocalWorkerConfig); err != nil {
			return WorkerConfiguration{}, time.Time{}, err
		}
		return conf, info.ModTime(), nil
	}
	return WorkerConfiguration{}, time.Time{}, err
}
