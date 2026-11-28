package api

import (
	"coordinator/internal/app"
	"coordinator/internal/domain"
	"coordinator/internal/utility"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

// HTTP to WebSocket updgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// sshDialTimeout is how long the coordinator waits for the worker's SSH port
const sshDialTimeout = 10 * time.Second

func LoadSSHRouter(router gin.IRouter, app *app.App) {
	auth := router.Group("", bearerAuthMiddleware(app, domain.UserRoleAdministrator))

	auth.GET("/:workerId", func(c *gin.Context) {
		rawID := c.Param("workerId")
		slog.Info("Received worker SSH terminal request", "workerId", rawID)

		workerID, err := strconv.Atoi(rawID)
		if err != nil {
			slog.Error("Invalid worker ID", "workerId", rawID, "error", err)
			utility.SendError(c, ErrInvalidWorkerID)
			return
		}

		ctx := c.Request.Context()
		worker, err := app.Worker.GetWorkerByID(ctx, workerID)
		if err != nil {
			slog.Error("Failed to fetch worker by ID", "workerId", workerID, "error", err)
			utility.SendError(c, err)
			return
		}

		slog.Info("Reading SSH private key", "workerId", workerID, "path", worker.Config.SSHKey)
		keyBytes, err := os.ReadFile(worker.Config.SSHKey)
		if err != nil {
			slog.Error("Failed to read SSH key file", "workerId", workerID, "path", worker.Config.SSHKey, "error", err)
			utility.SendError(c, err)
			return
		}

		signer, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			slog.Error("Failed to parse SSH private key", "workerId", workerID, "error", err)
			utility.SendError(c, err)
			return
		}

		pubKey, err := ssh.NewPublicKey(worker.PublicKey)
		if err != nil {
			slog.Error("Failed to parse worker host public key", "workerId", workerID, "error", err)
			utility.SendError(c, err)
			return
		}

		config := &ssh.ClientConfig{
			User:              worker.Config.SSHUser,
			Auth:              []ssh.AuthMethod{ssh.PublicKeys(signer)},
			HostKeyAlgorithms: []string{ssh.KeyAlgoED25519},
			HostKeyCallback:   ssh.FixedHostKey(pubKey),
			Timeout:           sshDialTimeout,
		}

		addr := net.JoinHostPort(worker.Config.Host, strconv.Itoa(int(worker.Config.SSHPort)))
		slog.Info("Dialing worker SSH", "workerId", workerID, "addr", addr, "user", worker.Config.SSHUser)
		client, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			slog.Error("Failed to dial worker SSH", "workerId", workerID, "addr", addr, "error", err)
			utility.SendError(c, err)
			return
		}
		defer client.Close()

		slog.Info("Creating SSH session", "workerId", workerID)
		session, err := client.NewSession()
		if err != nil {
			slog.Error("Failed to create SSH session", "workerId", workerID, "error", err)
			utility.SendError(c, err)
			return
		}
		defer session.Close()

		stdin, err := session.StdinPipe()
		if err != nil {
			slog.Error("Failed to open SSH stdin pipe", "workerId", workerID, "error", err)
			utility.SendError(c, err)
			return
		}

		stdout, err := session.StdoutPipe()
		if err != nil {
			slog.Error("Failed to open SSH stdout pipe", "workerId", workerID, "error", err)
			utility.SendError(c, err)
			return
		}

		session.Stderr = session.Stdout

		modes := ssh.TerminalModes{
			ssh.ECHO:          1,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}

		slog.Info("Requesting PTY", "workerId", workerID)
		if err := session.RequestPty("xterm-256color", 24, 80, modes); err != nil {
			slog.Error("Failed to request PTY", "workerId", workerID, "error", err)
			utility.SendError(c, err)
			return
		}

		slog.Info("Starting remote shell", "workerId", workerID)
		if err := session.Start("PATH=$HOME/rbin exec /bin/bash --restricted --noprofile --norc"); err != nil {
			slog.Error("Failed to start remote shell", "workerId", workerID, "error", err)
			utility.SendError(c, err)
			return
		}

		slog.Info("Upgrading connection to WebSocket", "workerId", workerID)
		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			slog.Error("Failed to upgrade connection to WebSocket", "workerId", workerID, "error", err)
			return
		}
		defer ws.Close()

		slog.Info("WebSocket SSH bridge established", "workerId", workerID)

		go func() {
			defer ws.Close()
			buf := make([]byte, 1024)
			for {
				n, err := stdout.Read(buf)
				if n > 0 {
					if writeErr := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
						slog.Error("Failed to write SSH output to WebSocket", "workerId", workerID, "error", writeErr)
						return
					}
				}
				if err != nil {
					if errors.Is(err, io.EOF) {
						slog.Info("SSH stdout closed (EOF)", "workerId", workerID)
					} else {
						slog.Error("Failed to read from SSH stdout", "workerId", workerID, "error", err)
					}
					return
				}
			}
		}()

		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				slog.Info("WebSocket read loop ended", "workerId", workerID, "error", err)
				break
			}
			if _, err := stdin.Write(msg); err != nil {
				slog.Error("Failed to write WebSocket input to SSH stdin", "workerId", workerID, "error", err)
				break
			}
		}

		slog.Info("Closed worker SSH terminal session", "workerId", workerID)
	})
}
