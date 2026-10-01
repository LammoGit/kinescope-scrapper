package storage

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type Job struct {
	JobID      string    `json:"job_id"`
	WorkerID   string    `json:"worker_id"`
	Issue      time.Time `json:"issue"`
	Expiration time.Time `json:"expiration"`
	From       uint      `json:"from"`
	To         uint      `json:"to"`
	Progress   uint      `json:"progress"`
	Attempt    uint      `json:"attempt"`
	Items      []uint    `json:"items"`
	Events     []string  `json:"events"`
	Status     string    `json:"status"`
	Paused     bool      `json:"paused"`
}

type ServerInfo struct {
	// General server fields
	Name string `json:"name"`
	Host string `json:"host"`

	// SSH fields
	SSHPort uint16 `json:"ssh_port"`
	SSHUser string `json:"ssh_user"`
	SSHKey  string `json:"ssh_key"`

	// HTTP server fields
	HTTPPort uint16 `json:"http_port"`

	// Server identity field
	PublicKey []byte `json:"public_key"` // ed25519 is assumed

	// Resource fields
	Threads        uint   `json:"threads"`
	ServersPath    string `json:"storage_path"`
	MinFreeSpaceGB uint   `json:"min_free_space_gb"`
}

type ServerMetrics struct {
	SpaceUsedBytes  uint64   `json:"space_used_bytes"`
	FreeSpaceBytes  uint64   `json:"free_space_bytes"`
	Status          uint     `json:"status"`
	From            uint     `json:"from"`
	To              uint     `json:"to"`
	Current         uint     `json:"current"`
	Found           uint     `json:"found"`
	Downloaded      uint     `json:"downloaded"`
	Errors          uint     `json:"errors"`
	DownloadSpeed   float64  `json:"download_speed"`
	ActiveChecks    uint     `json:"active_checks"`
	ActiveDownloads []string `json:"active_downloads"`
	CPULoad         float64  `json:"cpu_load"`
	RAMUsage        float64  `json:"ram_usage"`
	NetworkUsage    float64  `json:"network_usage"`
	SystemLoad      float64  `json:"system_load"`
	Uptime          uint64   `json:"uptime"`
}

type Server struct {
	mu      sync.RWMutex
	Info    ServerInfo    `json:"info"`
	Metrics ServerMetrics `json:"metrics"`
}

type Servers struct {
	mu      sync.RWMutex
	Servers map[string]*Server `json:"servers"`
}

func (st *Servers) GetWorkers() map[string]ServerInfo {
	st.mu.RLock()
	defer st.mu.RUnlock()

	snapshot := make(map[string]ServerInfo, len(st.Servers))

	for key, serverPtr := range st.Servers {
		if serverPtr == nil {
			continue
		}

		serverPtr.mu.Lock()
		snapshot[key] = serverPtr.Info
		serverPtr.mu.Unlock()
	}

	return snapshot
}

func (st *Servers) GetWorkerInfo(workerID string) (ServerInfo, bool) {
	st.mu.RLock()
	server, ok := st.Servers[workerID]
	st.mu.RUnlock()

	if !ok {
		return ServerInfo{}, false
	}

	server.mu.Lock()
	infoCopy := server.Info
	server.mu.Unlock()

	return infoCopy, true
}

func (st *Servers) GetWorkerMetrics(workerID string) (ServerMetrics, bool) {
	st.mu.RLock()
	server, ok := st.Servers[workerID]
	st.mu.RUnlock()

	if !ok {
		return ServerMetrics{}, false
	}

	server.mu.Lock()
	metricsCopy := server.Metrics
	server.mu.Unlock()

	return metricsCopy, true
}

func (st *Servers) UpdateServerMetrics(workerID string, metrics ServerMetrics) error {
	st.mu.RLock()
	server, ok := st.Servers[workerID]
	st.mu.RUnlock()

	if !ok {
		return fmt.Errorf("there is no worker: %s", workerID)
	}

	server.mu.Lock()
	server.Metrics = metrics
	server.mu.Unlock()

	return nil
}

func (st *Servers) UpdateServerInfo(workerID string, info ServerInfo) {}

func generateSSHKeyPair(privateKeyPath string) (string, error) {
	if err := os.MkdirAll(filepath.Dir(privateKeyPath), 0700); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}

	pemBlock, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return "", fmt.Errorf("marshal private key: %w", err)
	}
	if err := os.WriteFile(privateKeyPath, pem.EncodeToMemory(pemBlock), 0600); err != nil {
		return "", fmt.Errorf("write private key: %w", err)
	}

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return "", fmt.Errorf("marshal public key: %w", err)
	}

	return string(ssh.MarshalAuthorizedKey(sshPub)), nil
}

func (st *Servers) RegisterServer(info ServerInfo) (workerID, sshKey string, err error) {
	st.mu.Lock()
	defer st.mu.Unlock()

	for i := 1; ; i++ {
		name := fmt.Sprintf("worker-%03d", i)
		if _, ok := st.Servers[name]; !ok {
			workerID = name
			break
		}
	}

	privateKeyPath := filepath.Join("./keys", workerID)
	sshKey, err = generateSSHKeyPair(privateKeyPath)
	if err != nil {
		return "", "", err
	}

	info.Name = workerID
	info.SSHKey = privateKeyPath

	st.Servers[workerID] = &Server{
		Info: info,
	}

	return workerID, sshKey, nil
}
