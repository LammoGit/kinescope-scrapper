package domain

import "crypto/ed25519"

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
	SSHPort        int    `json:"ssh_port" form:"ssh_port" binding:"required,port"`
	SSHUser        string `json:"ssh_user" form:"ssh_user" binding:"required"`
	SSHKey         string `json:"ssh_key" form:"ssh_key" binding:"required"`
	Threads        int    `json:"threads" form:"threads" binding:"required,gt=0"`
	StoragePath    string `json:"storage_path" form:"storage_path" binding:"required,filepath"`
	MinFreeSpaceGB int    `json:"min_free_space_gb" form:"min_free_space_gb" binding:"required,min=0"`
}

// Worker represents a registered worker and its current state
type Worker struct {
	ID        int               `json:"id" form:"id" binding:"required,min=1"`
	Config    WorkerConfig      `json:"config" form:"config" binding:"required"`
	Status    string            `json:"status" form:"status" binding:"required,oneof=idle off upgrading working"`
	PublicKey ed25519.PublicKey `json:"public_key" form:"public_key" binding:"required"`
}
