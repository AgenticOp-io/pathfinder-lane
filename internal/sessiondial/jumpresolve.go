package sessiondial

import (
	"fmt"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/jump"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
	"github.com/scottpeterman/pathfinderssh/internal/sshcore"
)

func jumpCredentialLookup(l Lookup) jump.CredentialLookup {
	if l == nil {
		return nil
	}
	return func(name string) (jump.Credential, error) {
		c, err := l(name)
		if err != nil {
			return jump.Credential{}, err
		}
		return jump.Credential{
			Username:      c.Username,
			Password:      c.Password,
			KeyPath:       c.KeyPath,
			KeyPassphrase: c.KeyPassphrase,
		}, nil
	}
}

func boundHopToJumpConfig(h jump.BoundHop) *sshcore.JumpConfig {
	jc := &sshcore.JumpConfig{
		Host:     h.Host,
		Port:     h.Port,
		Username:       firstNonEmpty(h.Credential.Username, ""),
		Password: h.Credential.Password,
		PrivateKeyPath: h.Credential.KeyPath,
		KeyPassphrase:  h.Credential.KeyPassphrase,
	}
	if jc.PrivateKeyPath == "" && h.Credential.Password == "" {
		jc.Password = h.Credential.Password
	}
	return jc
}

// ResolveJumpChain applies jump_hosts.yaml when the session has no explicit jump hop.
func ResolveJumpChain(n sessions.Node, lookup Lookup, logf func(string, ...any)) ([]*sshcore.JumpConfig, string) {
	if n.Jump.InUse() {
		return nil, ""
	}
	path := jump.DefaultConfigPath()
	if path == "" {
		return nil, ""
	}
	cfg, ok, err := jump.LoadConfig(path)
	if err != nil || !ok || len(cfg.Rules) == 0 {
		return nil, ""
	}
	res, err := jump.NewResolver(cfg, logf)
	if err != nil {
		if logf != nil {
			logf("[jump] config invalid: %v", err)
		}
		return nil, ""
	}
	d := jump.Device{
		Name:     firstNonEmpty(n.Name, n.Host),
		Addr:     n.Host,
		Platform: n.Vendor,
	}
	dec := res.Resolve(d)
	if dec.Path.IsDirect() {
		return nil, ""
	}
	jlookup := jump.CachedLookup(jumpCredentialLookup(lookup))
	bound, err := jump.Bind(dec.Path, jlookup)
	if err != nil {
		if logf != nil {
			logf("[jump] bind: %v", err)
		}
		return nil, dec.Path.String()
	}
	var chain []*sshcore.JumpConfig
	for _, h := range bound {
		chain = append(chain, boundHopToJumpConfig(h))
	}
	route := dec.Path.String()
	if logf != nil && len(chain) > 0 {
		logf("[jump] route %s for %s", route, n.Target())
	}
	return chain, route
}

func sessionJumpToConfig(n sessions.Node, lookup Lookup) ([]*sshcore.JumpConfig, error) {
	if !n.Jump.InUse() {
		return nil, nil
	}
	jumpCred, err := resolve(lookup, n.Jump.Credential)
	if err != nil {
		return nil, fmt.Errorf("jump host: %w", err)
	}
	j := &sshcore.JumpConfig{
		Host:     strings.TrimSpace(n.Jump.Host),
		Port:     n.Jump.Port,
		Username: firstNonEmpty(jumpCred.Username, n.Jump.Username),
	}
	if key := firstNonEmpty(jumpCred.KeyPath, n.Jump.KeyPath); key != "" && jumpCred.AuthType != sessions.AuthPassword {
		j.PrivateKeyPath = key
		j.KeyPassphrase = firstNonEmpty(jumpCred.KeyPassphrase, n.Jump.KeyPassphrase)
	} else {
		j.Password = firstNonEmpty(jumpCred.Password, n.Jump.Password)
	}
	return []*sshcore.JumpConfig{j}, nil
}
