package api

import (
	"coordinator/internal/app"
	"coordinator/internal/domain"
	"coordinator/internal/utility"
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
		workerID, err := strconv.Atoi(c.Param("workerId"))
		if err != nil {
			utility.SendError(c, ErrInvalidWorkerID)
			return
		}

		ctx := c.Request.Context()
		worker, err := app.Worker.GetWorkerByID(ctx, workerID)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		keyBytes, err := os.ReadFile(worker.Config.SSHKey)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		signer, err := ssh.NewSignerFromKey(keyBytes)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		pubKey, err := ssh.NewPublicKey(worker.PublicKey)
		if err != nil {
			utility.SendError(c, err)
			return
		}

		config := &ssh.ClientConfig{
			User: worker.Config.SSHUser,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: ssh.FixedHostKey(pubKey),
		}

		addr := net.JoinHostPort(worker.Config.Host, strconv.Itoa(worker.Config.SSHPort))
		client, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			utility.SendError(c, err)
			return
		}
		defer client.Close()

		session, err := client.NewSession()
		if err != nil {
			utility.SendError(c, err)
			return
		}
		defer session.Close()

		stdin, err := session.StdinPipe()
		if err != nil {
			utility.SendError(c, err)
			return
		}

		stdout, err := session.StdoutPipe()
		if err != nil {
			utility.SendError(c, err)
			return
		}

		session.Stderr = session.Stdout

		modes := ssh.TerminalModes{
			ssh.ECHO:          1,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}

		if err := session.RequestPty("xterm-256color", 24, 80, modes); err != nil {
			utility.SendError(c, err)
			return
		}

		if err := session.Shell(); err != nil {
			utility.SendError(c, err)
			return
		}

		ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			utility.SendError(c, err)
			return
		}
		defer ws.Close()

		go func() {
			defer ws.Close()
			buf := make([]byte, 1024)
			for {
				n, err := stdout.Read(buf)
				if n > 0 {
					if writeErr := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
						utility.SendError(c, writeErr)
						return
					}
				}
				if err != nil {
					utility.SendError(c, err)
					return
				}
			}
		}()

		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				break
			}
			if _, err := stdin.Write(msg); err != nil {
				break
			}
		}
	})
}
