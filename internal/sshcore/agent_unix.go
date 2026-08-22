//go:build !windows

package sshcore

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// agentAuth returns agent-backed auth when SSH_AUTH_SOCK points at a live agent.
func agentAuth() ssh.AuthMethod {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil
	}
	return ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
		conn, err := net.Dial("unix", socket)
		if err != nil {
			return nil, fmt.Errorf("ssh agent: %w", err)
		}
		return agent.NewClient(conn).Signers()
	})
}
