package services

import (
	"context"
	"coordinator/internal/utility"
	"crypto/ed25519"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ProvisioningService installs the worker key on a remote host over SSH
type ProvisioningService struct {
	timeout time.Duration
}

// NewProvisioningService creates a new ProvisioningService
func NewProvisioningService() (*ProvisioningService, error) {
	return &ProvisioningService{timeout: 15 * time.Second}, nil
}

// Setup connects to the worker host over SSH
// It returns the worker's public key
func (s *ProvisioningService) Setup(ctx context.Context, sshHost string, sshPort int, sshUser, password, storagePath string) (ed25519.PublicKey, error) {
	// IN DEVELOPMENT
	// Provisioning service must
	// - connect via SSH
	// - generate SSH ed25519 keys
	// - save SSH ed25519 public key in file using sshKey path
	// - upload agent zip archive with venv and code
	// - disarchivate it
	// - source venv
	// - run code
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	addr := net.JoinHostPort(sshHost, strconv.Itoa(sshPort))
	dialer := net.Dialer{Timeout: s.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, utility.NewInternalError(fmt.Sprintf("failed to connect to %s: %v", addr, err))
	}

	config := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         s.timeout,
	}

	clientConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		_ = conn.Close()
		return nil, utility.NewInternalError(fmt.Sprintf("failed to establish ssh connection to %s: %v", addr, err))
	}
	client := ssh.NewClient(clientConn, chans, reqs)
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return nil, utility.NewInternalError(fmt.Sprintf("failed to open ssh session: %v", err))
	}
	defer session.Close()

	keyPath := strings.TrimRight(storagePath, "/") + "/.worker_ed25519"
	command := fmt.Sprintf(
		`set -e; mkdir -p %s; if [ ! -f %s ]; then ssh-keygen -t ed25519 -N "" -f %s >/dev/null 2>&1; fi; cat %s.pub`,
		shellQuote(storagePath),
		shellQuote(keyPath),
		shellQuote(keyPath),
		shellQuote(keyPath),
	)

	out, err := session.Output(command)
	if err != nil {
		return nil, utility.NewInternalError(fmt.Sprintf("failed to provision worker key: %v", err))
	}

	pub, _, _, _, err := ssh.ParseAuthorizedKey(out)
	if err != nil {
		return nil, utility.NewInternalError(fmt.Sprintf("worker returned an invalid public key: %v", err))
	}
	cryptoPub, ok := pub.(ssh.CryptoPublicKey)
	if !ok {
		return nil, utility.NewInternalError("worker public key isn't a crypto key")
	}
	edPub, ok := cryptoPub.CryptoPublicKey().(ed25519.PublicKey)
	if !ok {
		return nil, utility.NewInternalError("worker public key isn't an ed25519 key")
	}
	return edPub, nil
}

// shellQuote quotes a string for safe use in a POSIX shell command.
func shellQuote(v string) string {
	return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'"
}
