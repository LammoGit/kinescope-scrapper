package domain

import "crypto/ed25519"

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
	SSHPort        int    `json:"ssh_port" form:"ssh_port" binding:"required,min=1,max=65535"`
	SSHUser        string `json:"ssh_user" form:"ssh_user" binding:"required"`
	SSHKey         string `json:"ssh_key" form:"ssh_key" binding:"required"`
	Threads        int    `json:"threads" form:"threads" binding:"min=0"`
	StoragePath    string `json:"storage_path" form:"storage_path" binding:"required,filepath"`
	MinFreeSpaceGB int    `json:"min_free_space_gb" form:"min_free_space_gb" binding:"min=0"`
}

// Worker represents a registered worker and its current state
type Worker struct {
	ID        int               `json:"id" form:"id" binding:"min=0"`
	Config    WorkerConfig      `json:"config" form:"config" binding:"required"`
	Status    string            `json:"status" form:"status" binding:"required,oneof=idle off upgrading working"`
	PublicKey ed25519.PublicKey `json:"public_key" form:"public_key" binding:"required"`
}

// WorkerID returns the identifier of the worker
func (w *Worker) WorkerID() int {
	return w.ID
}

// WorkerUpdate optional fields to be updated
type WorkerUpdate struct {
	Name           *string `json:"name" form:"name" binding:"omitempty,min=1,max=64"`
	Host           *string `json:"host" form:"host"`
	SSHPort        *int    `json:"ssh_port" form:"ssh_port" binding:"omitempty,min=1,max=65535"`
	SSHUser        *string `json:"ssh_user" form:"ssh_user"`
	SSHKey         *string `json:"ssh_key" form:"ssh_key"`
	Threads        *int    `json:"threads" form:"threads" binding:"omitempty,min=0"`
	StoragePath    *string `json:"storage_path" form:"storage_path" binding:"omitempty,filepath"`
	MinFreeSpaceGB *int    `json:"min_free_space_gb" form:"min_free_space_gb" binding:"omitempty,min=0"`
}

// Update applies the non-nil fields of an update to a worker
func (w *Worker) Update(uu WorkerUpdate) {
	if uu.Name != nil {
		w.Config.Name = *uu.Name
	}

	if uu.Host != nil {
		w.Config.Host = *uu.Host
	}

	if uu.SSHPort != nil {
		w.Config.SSHPort = *uu.SSHPort
	}

	if uu.SSHUser != nil {
		w.Config.SSHUser = *uu.SSHUser
	}

	if uu.SSHKey != nil {
		w.Config.SSHKey = *uu.SSHKey
	}

	if uu.Threads != nil {
		w.Config.Threads = *uu.Threads
	}

	if uu.StoragePath != nil {
		w.Config.StoragePath = *uu.StoragePath
	}

	if uu.MinFreeSpaceGB != nil {
		w.Config.MinFreeSpaceGB = *uu.MinFreeSpaceGB
	}
}
