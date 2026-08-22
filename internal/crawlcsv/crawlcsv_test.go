package crawlcsv

import (
	"strings"
	"testing"
)

func TestParseTemplate(t *testing.T) {
	rows, err := ParseBytes([]byte(Template))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("rows=%d", len(rows))
	}
	if rows[0].Host != "192.0.2.1" || rows[0].Name != "core-sw1" {
		t.Fatalf("%+v", rows[0])
	}
	g := GroupByFolder(rows)
	if len(g["seeds"]) < 2 {
		t.Fatalf("seeds=%d", len(g["seeds"]))
	}
}

func TestParseRequiresHost(t *testing.T) {
	_, err := ParseBytes([]byte("name,port\nx,22\n"))
	if err == nil || !strings.Contains(err.Error(), "host") {
		t.Fatalf("err=%v", err)
	}
}
