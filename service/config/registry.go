package config

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/binkynet/BinkyNet/model"
)

type Registry interface {
	// Get returns the configuration for a worker with given ID.
	Get(id string) (WorkerConfiguration, error)
}

// WorkerConfiguration file format for local worker configuration file.
type WorkerConfiguration struct {
	model.LocalConfiguration
}

// NewRegistry create a new Registry implementation.
func NewRegistry(folder, defaultTopicPrefix string) (Registry, error) {
	return &registry{
		folder:  folder,
		configs: make(map[string]WorkerConfiguration),
	}, nil
}

type registry struct {
	mutex   sync.Mutex
	folder  string
	configs map[string]WorkerConfiguration
}

// Get returns the configuration for a worker with given ID.
func (r *registry) Get(id string) (WorkerConfiguration, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if x, found := r.configs[id]; found {
		return x, nil
	}

	// Read file
	content, err := ioutil.ReadFile(filepath.Join(r.folder, id+".json"))
	if err != nil {
		return WorkerConfiguration{}, maskAny(err)
	}

	// Parse file
	var conf WorkerConfiguration
	if err := json.Unmarshal(content, &conf); err != nil {
		return WorkerConfiguration{}, maskAny(err)
	}

	// Save config
	r.configs[id] = conf

	return conf, nil
}
