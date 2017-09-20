package client

import (
	"context"

	"github.com/binkynet/BinkyNet/model"
)

// API (HTTPs) of the NetManager.
type API interface {
	// GetWorkerConfig requests the local worker configuration for a worker with given id.
	GetWorkerConfig(ctx context.Context, workerID string) (model.LocalConfiguration, error)
	// Return a list of registered workers
	GetWorkers(ctx context.Context) ([]WorkerInfo, error)
}

// WorkerInfo holds registration info for a specific worker.
type WorkerInfo struct {
	// Unique ID of the worker
	ID string `json:"id"`
	// Endpoint of the worker
	Endpoint string
}
