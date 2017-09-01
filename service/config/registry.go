package config

import (
	"encoding/json"
	"io/ioutil"
	"path"
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
	// Name of MQTT topic the local worker should listen on for its runtime control messages
	ControlTopic string `json:"control-topic,omitempty"`
	// Name of MQTT topic the local worker should publish event data on
	DataTopic string `json:"data-topic,omitempty"`
}

// NewRegistry create a new Registry implementation.
func NewRegistry(folder, defaultTopicPrefix string) (Registry, error) {
	return &registry{
		folder:             folder,
		defaultTopicPrefix: defaultTopicPrefix,
		configs:            make(map[string]WorkerConfiguration),
	}, nil
}

type registry struct {
	mutex              sync.Mutex
	folder             string
	defaultTopicPrefix string
	configs            map[string]WorkerConfiguration
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

	// Fill defaults
	if conf.ControlTopic == "" {
		conf.ControlTopic = path.Join(r.defaultTopicPrefix, id, "control")
	}
	if conf.DataTopic == "" {
		conf.DataTopic = path.Join(r.defaultTopicPrefix, id, "data")
	}

	// Save config
	r.configs[id] = conf

	return conf, nil
}
