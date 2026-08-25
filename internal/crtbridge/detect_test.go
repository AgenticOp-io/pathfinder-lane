package crtbridge

import "testing"

func TestProbeEnvMissingCRT(t *testing.T) {
	p := ProbeEnv(`C:\this-crt-config-does-not-exist`, t.TempDir())
	if p.SecureCRTFound {
		t.Fatal("missing Config should not count as SecureCRT")
	}
	if p.Installed {
		t.Fatal("empty home should not look installed")
	}
}
