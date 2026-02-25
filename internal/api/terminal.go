package api

// API_SOURCE

import (
	"net/http"
	"strconv"
	"sync/atomic"

	"pb-deployer/internal/logger"

	"github.com/gorilla/websocket"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/crypto/ssh"
)

var terminalActive atomic.Bool

var wsUpgrader = websocket.Upgrader{
	// Local-only tool — origin check is intentionally permissive
	CheckOrigin: func(r *http.Request) bool { return true },
}

func handleTerminal(c *core.RequestEvent) error {
	log := logger.GetAPILogger()

	// Single active session guard
	if !terminalActive.CompareAndSwap(false, true) {
		return c.JSON(http.StatusConflict, map[string]any{
			"error": "A terminal session is already active",
		})
	}
	defer terminalActive.Store(false)

	host := c.Request.URL.Query().Get("host")
	user := c.Request.URL.Query().Get("user")
	portStr := c.Request.URL.Query().Get("port")
	if host == "" || user == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": "host and user query params are required",
		})
	}

	port := 22
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
			port = p
		}
	}

	// Upgrade to WebSocket before opening SSH (client is waiting)
	ws, err := wsUpgrader.Upgrade(c.Response, c.Request, nil)
	if err != nil {
		log.Error("WebSocket upgrade failed: %v", err)
		return nil // headers already sent
	}
	defer ws.Close()

	// Create and connect SSH client
	sshClient, err := createSSHClient(host, port, user)
	if err != nil {
		writeWSError(ws, "Failed to create SSH client: "+err.Error())
		return nil
	}
	defer sshClient.Close()

	if err := sshClient.Connect(); err != nil {
		writeWSError(ws, "SSH connection failed: "+err.Error())
		return nil
	}

	session, err := sshClient.NewSession()
	if err != nil {
		writeWSError(ws, "Failed to create SSH session: "+err.Error())
		return nil
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", 40, 220, modes); err != nil {
		writeWSError(ws, "Failed to request PTY: "+err.Error())
		return nil
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		writeWSError(ws, "Failed to open stdin pipe: "+err.Error())
		return nil
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		writeWSError(ws, "Failed to open stdout pipe: "+err.Error())
		return nil
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		writeWSError(ws, "Failed to open stderr pipe: "+err.Error())
		return nil
	}

	if err := session.Shell(); err != nil {
		writeWSError(ws, "Failed to start shell: "+err.Error())
		return nil
	}

	log.Info("Terminal session started for %s@%s", user, host)

	done := make(chan struct{})

	// Browser -> SSH stdin
	go func() {
		defer close(done)
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				return
			}
			if _, err := stdin.Write(msg); err != nil {
				return
			}
		}
	}()

	// SSH stdout -> browser
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				ws.WriteMessage(websocket.BinaryMessage, buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	// SSH stderr -> browser (merge into same stream)
	go func() {
		buf := make([]byte, 4*1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				ws.WriteMessage(websocket.BinaryMessage, buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	<-done
	log.Info("Terminal session closed for %s@%s", user, host)
	return nil
}

func writeWSError(ws *websocket.Conn, msg string) {
	ws.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[31mError: "+msg+"\x1b[0m\r\n"))
}
