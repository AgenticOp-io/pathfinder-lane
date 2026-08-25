package crtbridge

import (
	"strings"
	"testing"
)

func TestLaunchdPlistEscapes(t *testing.T) {
	got := launchdPlist([]string{`/Users/a&b/pflane`, "serve"})
	if !strings.Contains(got, launchdLabel) || !strings.Contains(got, "serve") {
		t.Fatal(got)
	}
	if !strings.Contains(got, "/Users/a&amp;b/pflane") {
		t.Fatal("path not escaped:", got)
	}
	if strings.Contains(got, "/Users/a&b/pflane") {
		t.Fatal("raw ampersand left in plist")
	}
}

func TestSystemdUnitQuotesSpaces(t *testing.T) {
	got := systemdUnit(`/opt/Pathfinder CRT/pflane`, []string{"serve"})
	if !strings.Contains(got, `" /opt/Pathfinder CRT/pflane"`) && !strings.Contains(got, `"/opt/Pathfinder CRT/pflane"`) {
		t.Fatal(got)
	}
	if !strings.Contains(got, " serve") {
		t.Fatal(got)
	}
}

func TestXmlEscape(t *testing.T) {
	if xmlEscape(`a<b>"c"`) != `a&lt;b&gt;&quot;c&quot;` {
		t.Fatal(xmlEscape(`a<b>"c"`))
	}
}
