package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	mrand "math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type AgentStatus uint

const (
	AgentStatusIdle AgentStatus = iota
	AgentStatusWorking
	AgentStatusFinished
)

type Metrics struct {
	SpaceUsedBytes  uint64      `json:"space_used_bytes"`
	FreeSpaceBytes  uint64      `json:"free_space_bytes"`
	Status          AgentStatus `json:"status"`
	From            uint        `json:"from"`
	To              uint        `json:"to"`
	Current         uint        `json:"current"`
	Found           uint        `json:"found"`
	Downloaded      uint        `json:"downloaded"`
	Errors          uint        `json:"errors"`
	DownloadSpeed   float64     `json:"download_speed"`
	ActiveChecks    uint        `json:"active_checks"`
	ActiveDownloads []string    `json:"active_downloads"`
	CPULoad         float64     `json:"cpu_load"`
	RAMUsage        float64     `json:"ram_usage"`
	NetworkUsage    float64     `json:"network_usage"`
	SystemLoad      float64     `json:"system_load"`
	Uptime          uint64      `json:"uptime"`
}

type RegisterRequest struct {
	Name           string `json:"name"`
	Host           string `json:"host"`
	SSHPort        uint16 `json:"ssh_port"`
	SSHUser        string `json:"ssh_user"`
	HTTPPort       uint16 `json:"http_port"`
	Threads        uint   `json:"threads"`
	StoragePath    string `json:"storage_path"`
	MinFreeSpaceGB uint   `json:"min_free_space_gb"`
	PublicKey      []byte `json:"public_key"`
}

type RegisterResponse struct {
	WorkerID string `json:"worker_id"`
	SSHKey   string `json:"ssh_key"`
}

type LeaseResponse struct {
	Name string `json:"name"`
	From uint   `json:"from"`
	To   uint   `json:"to"`
}

type Agent struct {
	mu                 sync.RWMutex
	coordinatorAddress string
	heartbeatInterval  time.Duration
	client             *http.Client
	ctx                context.Context
	cancel             func()

	// Identity
	workerID  string
	sshKey    string
	publicKey ed25519.PublicKey
	privKey   ed25519.PrivateKey

	// Job
	jobID   string
	current uint

	// Config
	Name           string `json:"name"`
	Host           string `json:"host"`
	SSHPort        uint16 `json:"ssh_port"`
	SSHUser        string `json:"ssh_user"`
	HTTPPort       uint16 `json:"http_port"`
	Threads        uint   `json:"threads"`
	StoragePath    string `json:"storage_path"`
	MinFreeSpaceGB uint   `json:"min_free_space_gb"`

	// Stats
	stats Metrics
}

var (
	coordinatorAddr = flag.String("coordinatorAddr", "127.0.0.1:8080", "coordinator address (host:port)")
	name            = flag.String("name", "worker", "agent display name")
	threads         = flag.Uint("threads", 8, "number of concurrent download threads")
	host            = flag.String("host", "127.0.0.1", "public host of this agent")
	sshPort         = flag.Uint("sshPort", 22, "SSH port")
	sshUser         = flag.String("sshUser", "root", "SSH user")
	httpPort        = flag.Uint("httpPort", 8081, "HTTP port advertised to the coordinator")
	storagePath     = flag.String("storagePath", "/tmp/videos", "storage path")
	minFreeSpaceGB  = flag.Uint("minFreeSpaceGB", 10, "minimum free space in GB")
)

func main() {
	flag.Parse()

	pub, priv, err := ed25519.GenerateKey(crand.Reader)
	if err != nil {
		log.Fatalf("key setup failed: %v", err)
	}

	a := &Agent{
		coordinatorAddress: *coordinatorAddr,
		heartbeatInterval:  5 * time.Second,
		client:             &http.Client{Timeout: 10 * time.Second},
		Name:               *name,
		Host:               *host,
		SSHPort:            uint16(*sshPort),
		SSHUser:            *sshUser,
		HTTPPort:           uint16(*httpPort),
		Threads:            *threads,
		StoragePath:        *storagePath,
		MinFreeSpaceGB:     *minFreeSpaceGB,
		privKey:            priv,
		publicKey:          pub,
		stats: Metrics{
			Status:         AgentStatusIdle,
			FreeSpaceBytes: uint64(*minFreeSpaceGB) * 1024 * 1024 * 1024,
		},
	}

	a.ctx, a.cancel = context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutting down...")
		a.cancel()
	}()

	a.startLifecycle()
}

func (a *Agent) signedPost(url string, body []byte, register bool) (*http.Response, error) {
	tsStr := strconv.FormatInt(time.Now().UnixNano(), 10)

	var prefix string
	if register {
		prefix = tsStr + ":"
	} else {
		prefix = tsStr + ":" + a.workerID + ":"
	}

	signed := make([]byte, 0, len(prefix)+len(body))
	signed = append(signed, prefix...)
	signed = append(signed, body...)

	sig := ed25519.Sign(a.privKey, signed)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Timestamp", tsStr)
	req.Header.Set("X-Signature", base64.StdEncoding.EncodeToString(sig))

	return a.client.Do(req)
}

func (a *Agent) startLifecycle() {
	stopChan := a.ctx.Done()

	regTicker := time.NewTicker(2 * time.Second)
	defer regTicker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-regTicker.C:
			if err := a.Register(); err == nil {
				log.Printf("agent %q registered as %q", a.Name, a.workerID)
				goto runLoop
			} else {
				log.Printf("registration failed: %v, retrying...", err)
			}
		}
	}

runLoop:
	go a.work()
	go a.leaseLoop()
	go a.progressLoop()

	hbTicker := time.NewTicker(a.heartbeatInterval)
	defer hbTicker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-hbTicker.C:
			a.sendHeartbeat()
		}
	}
}

func (a *Agent) Register() error {
	url := fmt.Sprintf("http://%s/api/workers/register", a.coordinatorAddress)

	reqBody := RegisterRequest{
		Name:           a.Name,
		Host:           a.Host,
		SSHPort:        a.SSHPort,
		SSHUser:        a.SSHUser,
		HTTPPort:       a.HTTPPort,
		Threads:        a.Threads,
		StoragePath:    a.StoragePath,
		MinFreeSpaceGB: a.MinFreeSpaceGB,
		PublicKey:      a.publicKey,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := a.signedPost(url, jsonData, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %s, body: %s", resp.Status, body)
	}

	var regResp RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return fmt.Errorf("failed to decode register response: %w", err)
	}
	if regResp.WorkerID == "" {
		return fmt.Errorf("coordinator returned empty worker_id")
	}

	a.mu.Lock()
	a.workerID = regResp.WorkerID
	a.sshKey = regResp.SSHKey
	a.mu.Unlock()

	return nil
}

func (a *Agent) sendHeartbeat() {
	a.mu.RLock()
	workerID := a.workerID
	jsonData, err := json.Marshal(a.stats)
	a.mu.RUnlock()

	if err != nil || workerID == "" {
		return
	}

	url := fmt.Sprintf("http://%s/api/workers/%s/heartbeat", a.coordinatorAddress, workerID)
	resp, err := a.signedPost(url, jsonData, false)
	if err != nil {
		log.Printf("heartbeat send error: %v", err)
		return
	}
	resp.Body.Close()
}

func (a *Agent) leaseLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.mu.RLock()
			idle := a.stats.Status == AgentStatusIdle
			a.mu.RUnlock()

			if idle {
				a.requestLease()
			}
		}
	}
}

func (a *Agent) progressLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.sendProgress()
		}
	}
}

func (a *Agent) requestLease() {
	a.mu.RLock()
	workerID := a.workerID
	a.mu.RUnlock()
	if workerID == "" {
		return
	}

	url := fmt.Sprintf("http://%s/api/workers/%s/lease", a.coordinatorAddress, workerID)

	body := []byte("{}")
	resp, err := a.signedPost(url, body, false)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	var lease LeaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&lease); err != nil {
		return
	}

	a.mu.Lock()
	a.current = lease.From
	a.stats.From = lease.From
	a.stats.To = lease.To
	a.stats.Current = lease.From
	a.stats.Found = 0
	a.stats.Downloaded = 0
	a.stats.Errors = 0
	a.stats.Status = AgentStatusWorking
	a.mu.Unlock()

	log.Printf("leased job %s [%d, %d]", lease.Name, lease.From, lease.To)
}

func (a *Agent) sendProgress() {
	a.mu.Lock()
	status := a.stats.Status
	if status == AgentStatusIdle || a.workerID == "" {
		a.mu.Unlock()
		return
	}

	jsonData, err := json.Marshal(a.stats)
	jobID := a.jobID

	if status == AgentStatusFinished {
		a.stats.Status = AgentStatusIdle
		a.jobID = ""
	}
	a.mu.Unlock()

	if err != nil {
		return
	}

	progressURL := fmt.Sprintf("http://%s/api/workers/%s/progress", a.coordinatorAddress, a.workerID)
	if resp, err := a.signedPost(progressURL, jsonData, false); err == nil {
		resp.Body.Close()
	}

	if status == AgentStatusFinished {
		completeURL := fmt.Sprintf("http://%s/api/workers/%s/complete", a.coordinatorAddress, a.workerID)
		if resp, err := a.signedPost(completeURL, jsonData, false); err == nil {
			resp.Body.Close()
		}
		log.Printf("completed job %s", jobID)
	}
}

func (a *Agent) work() {
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	ch := a.ctx.Done()
	startTime := time.Now()

	for {
		select {
		case <-ch:
			return
		case <-t.C:
			a.mu.Lock()

			a.stats.Uptime = uint64(time.Since(startTime).Seconds())

			if a.stats.Status != AgentStatusWorking {
				a.stats.CPULoad = mrand.Float64() * 5.0
				a.stats.RAMUsage = 15.0 + mrand.Float64()*2.0
				a.stats.NetworkUsage = 0.0
				a.stats.DownloadSpeed = 0.0
				a.stats.ActiveDownloads = []string{}
				a.stats.ActiveChecks = 0
				a.mu.Unlock()
				continue
			}

			if a.stats.Current > a.stats.To {
				a.stats.Status = AgentStatusFinished
				a.stats.DownloadSpeed = 0.0
				a.stats.ActiveDownloads = []string{}
				a.stats.ActiveChecks = 0
				a.mu.Unlock()
				continue
			}

			simulatedThreads := a.Threads
			if simulatedThreads == 0 {
				simulatedThreads = 1
			}

			a.stats.ActiveChecks = simulatedThreads
			a.stats.ActiveDownloads = []string{}

			for i := uint(0); i < simulatedThreads && a.current <= a.stats.To; i++ {
				fileURL := fmt.Sprintf("https://example.com/%d", a.current)
				a.stats.ActiveDownloads = append(a.stats.ActiveDownloads, fileURL)
				a.stats.Found++

				if mrand.Float64() < 0.05 {
					a.stats.Errors++
				} else {
					a.stats.Downloaded++
					a.stats.SpaceUsedBytes += 25 * 1024 * 1024
					if a.stats.FreeSpaceBytes > 25*1024*1024 {
						a.stats.FreeSpaceBytes -= 25 * 1024 * 1024
					}
				}

				a.current++
				a.stats.Current = a.current
			}

			a.stats.CPULoad = 40.0 + mrand.Float64()*35.0
			a.stats.RAMUsage = 55.0 + mrand.Float64()*15.0
			a.stats.DownloadSpeed = 5.0 + mrand.Float64()*45.0
			a.stats.NetworkUsage = a.stats.DownloadSpeed * 8.0
			a.stats.SystemLoad = a.stats.CPULoad / 100.0

			a.mu.Unlock()
		}
	}
}

func (a *Agent) Stop() {
	a.cancel()
}
