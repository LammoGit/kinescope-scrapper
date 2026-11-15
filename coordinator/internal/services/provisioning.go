package services

import (
	"context"
	"coordinator/internal/utility"
	"crypto/ed25519"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/povsister/scp"
	"golang.org/x/crypto/ssh"
)

// Errors
var (
	ErrEmptyCoordinatorPublicKey = utility.NewInternalError("Coordinator key mustn't be empty")
	ErrFailedToConnectViaSSH     = utility.NewConflictError("Failed to connect via ssh using provided data")
	ErrFailedToEstablishSession  = utility.NewInternalError("Failed to establish an ssh session")
	ErrFailedToGenerateSSHKeys   = utility.NewInternalError("Failed to generate keys")
	ErrFailedToSaveKey           = utility.NewInternalError("Failed to save generated keys")
	ErrFailedToAddTrusted        = utility.NewInternalError("Faield to add coordinator as a trusted client on the worker")
	ErrFailedToConnectViaSCP     = utility.NewConflictError("Failed to connect via scp using provided data")
	ErrFailedToStartAgent        = utility.NewInternalError("Failed to start an agent on the worker")
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
func (s *ProvisioningService) Setup(ctx context.Context, sshHost string, sshPort int, sshUser, password, sshKey string) (ed25519.PublicKey, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// connect via SSH
	config := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	addr := net.JoinHostPort(sshHost, strconv.Itoa(sshPort))
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, ErrFailedToConnectViaSSH
	}

	// Generate SSH ed25519 keys
	keygenSession, err := client.NewSession()
	if err != nil {
		return nil, ErrFailedToEstablishSession
	}
	defer keygenSession.Close()

	cmd := `mkdir -p ~/.ssh && ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519 -N "" -q && cat ~/.ssh/id_ed25519.pub`

	out, err := keygenSession.Output(cmd)
	if err != nil {
		return nil, ErrFailedToGenerateSSHKeys
	}

	publicKey := strings.TrimSpace(string(out))
	edKey, err := parseToEd25519PublicKey([]byte(publicKey))
	if err != nil {
		return edKey, err
	}

	// Save SSH ed25519 public key in file using sshKey path
	err = os.WriteFile(sshKey, edKey, 0644)
	if err != nil {
		return edKey, ErrFailedToSaveKey
	}

	// Add coordinator as trusted
	authSession, err := client.NewSession()
	if err != nil {
		return edKey, ErrFailedToEstablishSession
	}
	defer authSession.Close()

	coordAuthKey := s.pubKey

	cmd = fmt.Sprintf(
		`mkdir -p ~/.ssh && chmod 700 ~/.ssh && touch ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && grep -qxF %q ~/.ssh/authorized_keys || echo %q >> ~/.ssh/authorized_keys`,
		coordAuthKey,
		coordAuthKey,
	)

	if err := authSession.Run(cmd); err != nil {
		return edKey, ErrFailedToAddTrusted
	}

	// Upload agent binary
	scpClient, err := scp.NewClientFromExistingSSH(client, nil)
	if err != nil {
		return edKey, ErrFailedToConnectViaSCP
	}
	defer scpClient.Close()

	localBinary := "./dist/agent"
	remoteDir := "/opt/agent"
	remoteBinary := "/opt/agent/agent"

	prepSession, err := client.NewSession()
	if err != nil {
		return edKey, ErrFailedToEstablishSession
	}
	_ = prepSession.Run(fmt.Sprintf("mkdir -p %q", remoteDir))
	prepSession.Close()

	err = scpClient.CopyFileToRemote(localBinary, remoteBinary, &scp.FileTransferOption{
		Context:      ctx,
		PreserveProp: true,
	})
	if err != nil {
		return edKey, fmt.Errorf("failed to upload agent binary: %w", err)
	}

	// Run agent
	runSession, err := client.NewSession()
	if err != nil {
		return edKey, ErrFailedToEstablishSession
	}
	defer runSession.Close()

	runCmd := fmt.Sprintf(
		`chmod +x %q && nohup %q > %q/agent.log 2>&1 < /dev/null &`,
		remoteBinary,
		remoteBinary,
		remoteDir,
	)
	if err := runSession.Run(runCmd); err != nil {
		return edKey, ErrFailedToStartAgent
	}

	return edKey, nil

}

// parseToEd25519PublicKey parses given bytes into ed25519 public key
func parseToEd25519PublicKey(authorizedKeyBytes []byte) (ed25519.PublicKey, error) {
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(authorizedKeyBytes)
	if err != nil {
		return nil, ErrFailedToGenerateSSHKeys
	}

	cryptoPubKey, ok := pubKey.(ssh.CryptoPublicKey)
	if !ok {
		return nil, ErrFailedToGenerateSSHKeys
	}

	edKey, ok := cryptoPubKey.CryptoPublicKey().(ed25519.PublicKey)
	if !ok {
		return nil, ErrFailedToGenerateSSHKeys
	}

	return edKey, nil
}
