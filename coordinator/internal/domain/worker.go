package domain

import (
	"context"
	"crypto/ed25519"
	"slices"
)

// IWorkerRepository stores registered workers
type IWorkerRepository interface {
	SaveWorker(ctx context.Context, worker *Worker) error
	GetWorkerByID(ctx context.Context, workerID int) (Worker, error)
	GetWorkers(ctx context.Context) ([]Worker, error)
	DeleteWorkerByID(ctx context.Context, workerID int) error
}

// Worker statuses
const (
	WorkerStatusIdle      string = "idle"
	WorkerStatusOff       string = "off"
	WorkerStatusUpgrading string = "upgrading"
	WorkerStatusWorking   string = "working"
)

// WorkerConfig contains everything needed to connect to and configure a worker
type WorkerConfig struct {
	Name           string `json:"name" form:"name" binding:"required,min=1,max=64"`
	Host           string `json:"host" form:"host" binding:"required"`
	SSHPort        uint16 `json:"ssh_port" form:"ssh_port" binding:"required,port"`
	SSHUser        string `json:"ssh_user" form:"ssh_user" binding:"required"`
	SSHKey         string `json:"ssh_key" form:"ssh_key" binding:"required,filepath"`
	Threads        int    `json:"threads" form:"threads" binding:"required,gt=0"`
	StoragePath    string `json:"storage_path" form:"storage_path" binding:"required,dirpath"`
	MinFreeSpaceGB int    `json:"min_free_space_gb" form:"min_free_space_gb" binding:"required,min=0"`
}

// Worker represents a registered worker and its current state
type Worker struct {
	ID        int               `json:"id" form:"id" binding:"required,min=1"`
	Config    WorkerConfig      `json:"config" form:"config" binding:"required"`
	Status    string            `json:"status" form:"status" binding:"required,oneof=idle off upgrading working"`
	PublicKey ed25519.PublicKey `json:"public_key" form:"public_key" binding:"required"`
}

// Clone creates a deep-copy of the calling WorkerConfig
func (wc WorkerConfig) Clone() WorkerConfig {
	return wc
}

// Clone creates a deep-copy of the calling Worker
func (w Worker) Clone() Worker {
	clone := w
	clone.Config = w.Config.Clone()
	clone.PublicKey = slices.Clone(w.PublicKey)
	return clone
}
