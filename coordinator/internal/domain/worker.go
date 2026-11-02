package domain

type Worker struct {
	Config WorkerConfig
	Status WorkerStatus
	ranges []*Range
}

type WorkerConfig struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Host           string `json:"host"`
	SSHPort        int    `json:"ssh_port"`
	SSHUser        string `json:"ssh_user"`
	SSHKey         string `json:"ssh_key"`
	Threads        int    `json:"threads"`
	StoragePath    string `json:"storage_path"`
	MinFreeSpaceGB int    `json:"min_free_space_gb"`
}

type WorkerStatus string

const (
	WorkerStatusIdle      WorkerStatus = "idle"
	WorkerStatusOff       WorkerStatus = "off"
	WorkerStatusUpgrading WorkerStatus = "upgrading"
	WorkerStatusWorking   WorkerStatus = "working"
)
