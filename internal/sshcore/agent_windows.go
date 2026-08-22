//go:build windows

package sshcore

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Microsoft/go-winio"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Windows OpenSSH agent default named pipe.
const windowsSSHAgentPipe = `\\.\pipe\openssh-ssh-agent`

// agentAuth dials the Windows OpenSSH agent named pipe (or SSH_AUTH_SOCK if set).
func agentAuth() ssh.AuthMethod {
	return ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
		conn, err := dialWindowsAgent()
		if err != nil {
			return nil, fmt.Errorf("ssh agent: %w", err)
		}
		return agent.NewClient(conn).Signers()
	})
}

func dialWindowsAgent() (net.Conn, error) {
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		// OpenSSH for Windows may set SSH_AUTH_SOCK to the pipe path.
		timeout := 2 * time.Second
		if c, err := winio.DialPipe(sock, &timeout); err == nil {
			return c, nil
		}
		if c, err := net.DialTimeout("unix", sock, timeout); err == nil {
			return c, nil
		}
	}
	timeout := 2 * time.Second
	return winio.DialPipe(windowsSSHAgentPipe, &timeout)
}
