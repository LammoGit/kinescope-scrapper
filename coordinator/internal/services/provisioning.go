package services

import (
	"context"
	"coordinator/internal/utility"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/povsister/scp"
	"golang.org/x/crypto/ssh"
)

// Errors
var (
	ErrEmptyCoordinatorPublicKey = utility.NewInternalError("coordinator key mustn't be empty")
	ErrFailedToConnectViaSSH     = utility.NewConflictError("failed to connect via ssh using provided data")
	ErrFailedToEstablishSession  = utility.NewInternalError("failed to establish an ssh session")
	ErrFailedToGenerateSSHKeys   = utility.NewInternalError("failed to generate keys")
	ErrFailedToSaveKey           = utility.NewInternalError("failed to save generated keys")
	ErrFailedToAddTrusted        = utility.NewInternalError("faield to add coordinator as a trusted client on the worker")
	ErrFailedToConnectViaSCP     = utility.NewConflictError("failed to connect via scp using provided data")
	ErrFailedToStartAgent        = utility.NewInternalError("failed to start an agent on the worker")
	ErrFailedToAddRBin           = utility.NewInternalError("failed to restrict available binaries")
)

// ProvisioningService installs the worker key on a remote host over SSH
type ProvisioningService struct {
	pubKey string
}

// NewProvisioningService creates a new ProvisioningService
func NewProvisioningService(pubKey string) (*ProvisioningService, error) {
	if pubKey == "" {
		return nil, ErrEmptyCoordinatorPublicKey
	}

	return &ProvisioningService{
		pubKey: pubKey,
	}, nil
}

// Setup connects to the worker host over SSH
// It returns the worker's public key
func (s *ProvisioningService) Setup(ctx context.Context, sshHost string, sshPort uint16, sshUser, password, sshKey string) (ed25519.PublicKey, error) {
	slog.Info("Starting provisioning")
	if err := ctx.Err(); err != nil {
		slog.Error("Context exited provisioning", "error", err)
		return nil, err
	}

	// Capture the worker's SSH server host key during handshake
	var workerHostKey ed25519.PublicKey
	hostKeyCallback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if cryptoKey, ok := key.(ssh.CryptoPublicKey); ok {
			if edPub, ok := cryptoKey.CryptoPublicKey().(ed25519.PublicKey); ok {
				workerHostKey = edPub
			}
		}
		return nil
	}

	// Connect via SSH
	slog.Info("Starting SSH connection")
	config := &ssh.ClientConfig{
		User:              sshUser,
		Auth:              []ssh.AuthMethod{ssh.Password(password)},
		HostKeyAlgorithms: []string{ssh.KeyAlgoED25519},
		HostKeyCallback:   hostKeyCallback,
		Timeout:           10 * time.Second,
	}
	addr := net.JoinHostPort(sshHost, strconv.Itoa(int(sshPort)))
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		slog.Error("Failed to connect via SSH", "addr", addr, "error", err)
		return nil, ErrFailedToConnectViaSSH
	}
	defer client.Close()

	if len(workerHostKey) == 0 {
		slog.Error("Worker did not present an ed25519 host key", "addr", addr)
		return nil, ErrFailedToGenerateSSHKeys
	}

	// Generate a ed25519 keypair for the coordinator to access this worker
	slog.Info("Generating worker-specific coordinator SSH keypair")
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		slog.Error("Failed to generate ed25519 keypair", "error", err)
		return nil, ErrFailedToGenerateSSHKeys
	}

	pemBlock, err := ssh.MarshalPrivateKey(privKey, fmt.Sprintf("scrappy-worker-%s", sshHost))
	if err != nil {
		slog.Error("Failed to marshal private key to OpenSSH PEM", "error", err)
		return nil, ErrFailedToGenerateSSHKeys
	}

	if err := os.MkdirAll(filepath.Dir(sshKey), 0700); err != nil {
		slog.Error("Failed to create key directory", "path", sshKey, "error", err)
		return nil, ErrFailedToSaveKey
	}

	if err := os.WriteFile(sshKey, pem.EncodeToMemory(pemBlock), 0600); err != nil {
		slog.Error("Failed to write worker-specific private key", "path", sshKey, "error", err)
		return nil, ErrFailedToSaveKey
	}

	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		slog.Error("Failed to convert public key to SSH format", "error", err)
		return nil, ErrFailedToGenerateSSHKeys
	}
	authorizedKeyLine := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPubKey)))

	// Add the worker-specific coordinator public key to worker's authorized_keys
	authSession, err := client.NewSession()
	if err != nil {
		slog.Error("Failed to create auth session", "error", err)
		return nil, ErrFailedToEstablishSession
	}
	defer authSession.Close()

	cmd := fmt.Sprintf(
		`mkdir -p ~/.ssh && chmod 700 ~/.ssh && touch ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && grep -qxF %q ~/.ssh/authorized_keys || echo %q >> ~/.ssh/authorized_keys`,
		authorizedKeyLine,
		authorizedKeyLine,
	)

	if out, err := authSession.CombinedOutput(cmd); err != nil {
		slog.Error("Failed to add coordinator's key to trusted", "error", err, "output", string(out))
		return nil, ErrFailedToAddTrusted
	}

	// Restrict available binaries
	rbinSession, err := client.NewSession()
	if err != nil {
		slog.Error("Failed to create auth session", "error", err)
		return nil, ErrFailedToEstablishSession
	}
	defer rbinSession.Close()

	cmd = `
mkdir -p ~/rbin && \
ln -sf /bin/ls ~/rbin/ls && \
ln -sf /bin/cd ~/rbin/cd && \
ln -sf /bin/cat ~/rbin/cat && \
ln -sf /usr/bin/tail ~/rbin/tail && \
ln -sf /bin/df ~/rbin/df
`

	if out, err := rbinSession.CombinedOutput(cmd); err != nil {
		slog.Error("Failed to create a folder for restricted binaries", "error", err, "output", string(out))
		return nil, ErrFailedToAddRBin
	}

	// Upload agent binary
	scpClient, err := scp.NewClientFromExistingSSH(client, &scp.ClientOption{})
	if err != nil {
		slog.Error("Failed to connect via SCP", "error", err)
		return nil, ErrFailedToConnectViaSCP
	}
	defer scpClient.Close()

	homeSession, err := client.NewSession()
	if err != nil {
		return nil, ErrFailedToEstablishSession
	}
	homeOut, err := homeSession.Output("echo $HOME")
	homeSession.Close()
	if err != nil {
		slog.Error("Failed to resolve remote HOME", "error", err)
		return nil, ErrFailedToEstablishSession
	}

	remoteDir := filepath.ToSlash(filepath.Join(strings.TrimSpace(string(homeOut)), ".local", "bin"))
	remoteBinary := filepath.ToSlash(filepath.Join(remoteDir, "agent"))

	prepSession, err := client.NewSession()
	if err != nil {
		return nil, ErrFailedToEstablishSession
	}
	if prepOut, err := prepSession.CombinedOutput(fmt.Sprintf("mkdir -p %q", remoteDir)); err != nil {
		prepSession.Close()
		slog.Error("Failed to create remote directory", "dir", remoteDir, "error", err, "output", string(prepOut))
		return nil, fmt.Errorf("failed to create remote dir: %w", err)
	}
	prepSession.Close()

	err = scpClient.CopyFileToRemote("./agent", remoteBinary, &scp.FileTransferOption{
		Context:      ctx,
		PreserveProp: true,
	})
	if err != nil {
		slog.Error("Failed to upload agent binary", "error", err)
		return nil, fmt.Errorf("failed to upload agent binary: %w", err)
	}

	// Run agent
	runSession, err := client.NewSession()
	if err != nil {
		return nil, ErrFailedToEstablishSession
	}
	defer runSession.Close()

	runCmd := fmt.Sprintf(
		`chmod +x %q && nohup %q > %q/agent.log 2>&1 < /dev/null &`,
		remoteBinary,
		remoteBinary,
		remoteDir,
	)
	if err := runSession.Run(runCmd); err != nil {
		slog.Error("Failed to start agent", "error", err)
		return nil, ErrFailedToStartAgent
	}

	return workerHostKey, nil
}
