// Mock agent implementation for testing purposes
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

// AgentStatus enumeration type representing status of the agent
type AgentStatus uint

const (
	AgentStatusIdle     AgentStatus = iota // Agent doesn't have a block
	AgentStatusWorking                     // Agent is currently working on a block
	AgentStatusFinished                    // Agent have finished its previous block and
)

// Metrics structure used for parsing and storing worker's metrics
type Metrics struct {
	SpaceUsedBytes  uint64      `json:"space_used_bytes"` // Amount of used up space
	FreeSpaceBytes  uint64      `json:"free_space_bytes"` // Amount of remaining free space
	Status          AgentStatus `json:"status"`           // Current agent status
	From            uint        `json:"from"`             // Start of the current block
	To              uint        `json:"to"`               // End of the current block
	Current         uint        `json:"current"`          // Current maximum item agent is working on
	Found           uint        `json:"found"`            // Number of found items
	Downloaded      uint        `json:"downloaded"`       // Number of downloaded items
	Errors          uint        `json:"errors"`           // Number of errors
	DownloadSpeed   float64     `json:"download_speed"`   // Current download speed
	ActiveChecks    uint        `json:"active_checks"`    // Number of active item checks
	ActiveDownloads []string    `json:"active_downloads"` // Number of active item downloads
	CPULoad         float64     `json:"cpu_load"`         // Current CPU load in percentages
	RAMUsage        float64     `json:"ram_usage"`        // Current RAM usage in percentages
	NetworkUsage    float64     `json:"network_usage"`    // Current network usage in bytes per second
	SystemLoad      float64     `json:"system_load"`      // Current system load in percentages
	Uptime          uint64      `json:"uptime"`           // Current uptime of the agent
}

// RegisterRequest structure used for parsing and storing registration request's body
type RegisterRequest struct {
	Name           string `json:"name"`              // Name of the agent (Probably should be removed as worker id is a name of the worker)
	Host           string `json:"host"`              // Public host of the agent (public ip or domain)
	SSHPort        uint16 `json:"ssh_port"`          // The port used by the SSH
	SSHUser        string `json:"ssh_user"`          // Name of the SSH user to be used by the coordinator
	HTTPPort       uint16 `json:"http_port"`         // HTTP port of the agent (Probably should remove and use websocket connection)
	Threads        uint   `json:"threads"`           // Number of threads used on an agent
	StoragePath    string `json:"storage_path"`      // Path for storing the downloaded files and meta data
	MinFreeSpaceGB uint   `json:"min_free_space_gb"` // Minimum allocated space on the worker for storing the files
	PublicKey      []byte `json:"public_key"`        // Ed25519 public key used for authentication
}

// RegisterResponse structure used for parsing and storing registration response's body
type RegisterResponse struct {
	WorkerID string `json:"worker_id"` // The worker id given after a registration
	SSHKey   string `json:"ssh_key"`   // The coordinator's public ssh key
}

// LeaseResponse structure used for parsing and storing lease response's body
type LeaseResponse struct {
	JobID string `json:"name"` // The block's parent job's id
	From  uint   `json:"from"` // Start of the item id range
	To    uint   `json:"to"`   // End of the item id range
}

type Agent struct {
	mu                 sync.RWMutex
	coordinatorAddress string          // Address of the coordinator
	heartbeatInterval  time.Duration   // Interval between heartbeats
	client             *http.Client    // HTTP client used to send requests
	ctx                context.Context // Global context for graceful shutdown
	shutdown           func()          // Cancel function of the global context

	// Identity
	workerID  string             // ID of the worker given by the coordinator after registration
	sshKey    string             // Public SSH key used by coordinator to connect via SSH
	publicKey ed25519.PublicKey  // Public ed25519 key used for authentication
	privKey   ed25519.PrivateKey // Private ed25519 key used for authentication

	// Job
	jobID   string // Current job's ID
	current uint   // Current item

	// Config
	Name           string `json:"name"`              // Name of the agent (Probably should be removed as worker id is a name of the worker)
	Host           string `json:"host"`              // Public host of the agent (public ip or domain)
	SSHPort        uint16 `json:"ssh_port"`          // The port used by the SSH
	SSHUser        string `json:"ssh_user"`          // Name of the SSH user to be used by the coordinator
	HTTPPort       uint16 `json:"http_port"`         // HTTP port of the agent (Probably should remove and use websocket connection)
	Threads        uint   `json:"threads"`           // Number of threads used on an agent
	StoragePath    string `json:"storage_path"`      // Path for storing the downloaded files and meta data
	MinFreeSpaceGB uint   `json:"min_free_space_gb"` // Minimum allocated space on the worker for storing the files

	// Stats
	stats Metrics // Current metrics of the agent
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
	// Parsing flags
	flag.Parse()

	// Generating ed25519 key pair
	pub, priv, err := ed25519.GenerateKey(crand.Reader)
	if err != nil {
		log.Fatalf("key setup failed: %v", err)
	}

	// Creating a new agent
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

	// Creating a global context and it's cancel function
	a.ctx, a.shutdown = context.WithCancel(context.Background())

	// Shutdown server on SIGINT and SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutting down...")
		a.Stop()
	}()

	a.startLifecycle()
}

// signedPost sends a signed post request to the given URL with the given body
func (a *Agent) signedPost(url string, body []byte, register bool) (*http.Response, error) {
	// Get number of nanoseconds of the current timestamp and covert it into a string
	tsStr := strconv.FormatInt(time.Now().UnixNano(), 10)

	// Generate prefix based on whether it's a registration
	var prefix string
	if register {
		prefix = tsStr + ":"
	} else {
		prefix = tsStr + ":" + a.workerID + ":"
	}

	// Generate byte sequence which will be signed
	signed := make([]byte, 0, len(prefix)+len(body))
	signed = append(signed, prefix...)
	signed = append(signed, body...)

	// Generate signature
	sig := ed25519.Sign(a.privKey, signed)

	// Create a POST request to the given url with the given body
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// Set `Content-Type` to JSON and timestamp with signature
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Timestamp", tsStr)
	req.Header.Set("X-Signature", base64.StdEncoding.EncodeToString(sig))

	// Send the request
	return a.client.Do(req)
}

// startLifecycle
func (a *Agent) startLifecycle() {
	// Channel that receives on shutdown
	stopChan := a.ctx.Done()

	// Ticker for repeated registration on failure
	regTicker := time.NewTicker(2 * time.Second)
	defer regTicker.Stop()

	for {
		select {
		case <-stopChan:
			// Exit lifecycle on shutdown
			return
		case <-regTicker.C:
			// Try to register on the registration ticker
			if err := a.Register(); err == nil {
				log.Printf("agent %q registered as %q", a.Name, a.workerID)
				goto runLoop // exit the registration loop
			} else {
				log.Printf("registration failed: %v, retrying...", err)
			}
		}
	}

runLoop:
	go a.work()         // Start the work simulation goroutine
	go a.leaseLoop()    // Start leasing goroutine
	go a.progressLoop() // Start progress sending goroutine

	// Ticker for the heartbeat
	hbTicker := time.NewTicker(a.heartbeatInterval)
	defer hbTicker.Stop()

	for {
		select {
		case <-stopChan:
			// Exit lifecycle on shutdown
			return
		case <-hbTicker.C:
			// Send a heartbeat on the heartbeat ticker
			a.sendHeartbeat()
		}
	}
}

// Register tries to register the agent on the coordinator
func (a *Agent) Register() error {
	// The coordinator registration endpoint
	url := fmt.Sprintf("http://%s/api/workers/register", a.coordinatorAddress)

	// Registration request body structure
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

	// Parsing registration body structure into a JSON string
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	// Sending a signed POST request to the registration endpoint
	resp, err := a.signedPost(url, jsonData, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// If response isn't successful read the error message and return it
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %s, body: %s", resp.Status, body)
	}

	// Decode received response into the standardized RegisterResponse structure
	var regResp RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return fmt.Errorf("failed to decode register response: %w", err)
	}
	if regResp.WorkerID == "" {
		return fmt.Errorf("coordinator returned empty worker_id")
	}

	// Write received from regestration settings while locking mutex for writing
	a.mu.Lock()
	a.workerID = regResp.WorkerID
	a.sshKey = regResp.SSHKey
	a.mu.Unlock()

	return nil
}

// sendHeartbeat sends heartbeat message to the coordinator
func (a *Agent) sendHeartbeat() {
	// Get worker data while locking mutex for reading
	a.mu.RLock()
	workerID := a.workerID
	a.mu.RUnlock()

	// Coordinator's heartbeat endpoint
	url := fmt.Sprintf("http://%s/api/workers/%s/heartbeat", a.coordinatorAddress, workerID)

	// Sending signed heartbeat message
	resp, err := a.signedPost(url, []byte{}, false)
	if err != nil {
		log.Printf("heartbeat send error: %v", err)
		return
	}
	resp.Body.Close()
}

// leaseLoop starts a loop for sending leasing requests
func (a *Agent) leaseLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			// Exit leasing loop on shutdown
			return
		case <-ticker.C:
			// Check if agent is idle while locking mutex for reading
			a.mu.RLock()
			idle := a.stats.Status == AgentStatusIdle
			a.mu.RUnlock()

			// If agent is currently idle send a lease request
			if idle {
				a.requestLease()
			}
		}
	}
}

// progressLoop starts a loop for sending progress messages
func (a *Agent) progressLoop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			// Exit progress loop on shutdown
			return
		case <-ticker.C:
			// Send metrics to the coordinator on ticker
			a.sendProgress()
		}
	}
}

// requestLease requests a new range block from the coordinator
func (a *Agent) requestLease() {
	// Read agent's workerID while locking mutex for reading
	a.mu.RLock()
	workerID := a.workerID
	a.mu.RUnlock()
	if workerID == "" {
		return
	}

	// Coordinator's leasing endpoint
	url := fmt.Sprintf("http://%s/api/workers/%s/lease", a.coordinatorAddress, workerID)

	// Sending a signed lease request
	resp, err := a.signedPost(url, []byte{}, false)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// If response ins't successful leave
	if resp.StatusCode != http.StatusOK {
		return
	}

	// Parse lease response's body
	var lease LeaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&lease); err != nil {
		return
	}

	// Update current metrics and job settings while locking mutex for writing
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

	log.Printf("leased job %s [%d, %d]", lease.JobID, lease.From, lease.To)
}

// sendProgress sends current agent's metrics to the coordinator
func (a *Agent) sendProgress() {
	// Lock mutex for writing
	a.mu.Lock()
	// If agent is currently idle or isn't registered leave
	status := a.stats.Status
	if status == AgentStatusIdle || a.workerID == "" {
		a.mu.Unlock()
		return
	}

	// Convert metrics to a JSON string
	jsonData, err := json.Marshal(a.stats)
	jobID := a.jobID

	// If agent is finished switch to idle
	if status == AgentStatusFinished {
		a.stats.Status = AgentStatusIdle
		a.jobID = ""
	}
	a.mu.Unlock()

	if err != nil {
		return
	}

	// Send a signed metrics message to the coordinator's progress endpoint
	progressURL := fmt.Sprintf("http://%s/api/workers/%s/progress", a.coordinatorAddress, a.workerID)
	if resp, err := a.signedPost(progressURL, jsonData, false); err == nil {
		resp.Body.Close()
	}

	// If current status is finished, send a request to the coordinator's completion endpoint
	if status == AgentStatusFinished {
		completeURL := fmt.Sprintf("http://%s/api/workers/%s/complete", a.coordinatorAddress, a.workerID)
		if resp, err := a.signedPost(completeURL, jsonData, false); err == nil {
			resp.Body.Close()
		}
		log.Printf("completed job %s", jobID)
	}
}

// work generates metrics pretending to be working
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

			// Metrics for not working agent
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

			// If current item is higher than the range requirement the work is finished
			if a.stats.Current > a.stats.To {
				a.stats.Status = AgentStatusFinished
				a.stats.DownloadSpeed = 0.0
				a.stats.ActiveDownloads = []string{}
				a.stats.ActiveChecks = 0
				a.mu.Unlock()
				continue
			}

			simulatedThreads := min(1, a.Threads)

			a.stats.ActiveChecks = simulatedThreads
			a.stats.ActiveDownloads = []string{}

			for i := uint(0); i < simulatedThreads && a.current <= a.stats.To; i++ {
				fileURL := fmt.Sprintf("http://kinescope.io/%d", a.current)
				a.stats.ActiveDownloads = append(a.stats.ActiveDownloads, fileURL)
				a.stats.Found++

				// 5% chance of error during download
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

// Stop gracefully shutdowns the agent
func (a *Agent) Stop() {
	a.shutdown()
}
