package sshcore

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

func jumpChain(c Config) []*JumpConfig {
	if len(c.JumpChain) > 0 {
		return c.JumpChain
	}
	if c.Jump != nil && c.Jump.Host != "" {
		return []*JumpConfig{c.Jump}
	}
	return nil
}

func dialThroughBastion(parent *ssh.Client, j *JumpConfig, c Config, hostKeyCB ssh.HostKeyCallback) (*ssh.Client, error) {
	port := j.Port
	if port == 0 {
		port = 22
	}
	addr := net.JoinHostPort(j.Host, fmt.Sprintf("%d", port))
	conn, err := parent.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("reach bastion %s through parent: %w", addr, err)
	}
	methods, err := jumpAuthMethods(j)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if len(methods) == 0 {
		conn.Close()
		return nil, fmt.Errorf("bastion %s: no usable credentials", addr)
	}
	conn.SetDeadline(time.Now().Add(c.Timeout))
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, &ssh.ClientConfig{
		User:              j.Username,
		Auth:              methods,
		HostKeyCallback:   hostKeyCB,
		Timeout:           c.Timeout,
		Config:            algorithmPolicy(c.LegacyAlgorithms),
		HostKeyAlgorithms: hostKeyAlgos(c.LegacyAlgorithms),
	})
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("SSH handshake with bastion %s: %w", addr, err)
	}
	conn.SetDeadline(time.Time{})
	return ssh.NewClient(sshConn, chans, reqs), nil
}

func dialJumpWith(j *JumpConfig, c Config, hostKeyCB ssh.HostKeyCallback) (*ssh.Client, error) {
	tmp := c
	tmp.Jump = j
	return dialJump(&tmp, hostKeyCB)
}

func jumpAuthMethods(j *JumpConfig) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod
	if j.PrivateKeyPath != "" {
		keyData, err := readKeyFile(j.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("bastion key: %w", err)
		}
		signer, err := parseSigner(keyData, j.KeyPassphrase)
		if err != nil {
			return nil, fmt.Errorf("bastion key: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	if j.Password != "" {
		methods = append(methods, ssh.Password(j.Password))
	}
	return methods, nil
}
