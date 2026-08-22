package scripts

import "testing"

func TestRecorderNotesAndUpsert(t *testing.T) {
	r := NewRecorder("Demo")
	r.Note("show ver\n")
	r.Note("exit\n")
	r.Stop()
	r.Note("ignored\n")
	s := r.Script()
	if s.Name != "Demo" || len(s.Steps) != 2 {
		t.Fatalf("got %+v", s)
	}
	f := Upsert(File{Version: 1}, s)
	if len(f.Scripts) != 1 || f.Scripts[0].Steps[0].Send != "show ver\n" {
		t.Fatalf("upsert %+v", f)
	}
	f = Upsert(f, Script{Name: "Demo", Steps: []Step{{Send: "x\n"}}})
	if len(f.Scripts) != 1 || f.Scripts[0].Steps[0].Send != "x\n" {
		t.Fatalf("replace %+v", f)
	}
}

func TestInferPrompt(t *testing.T) {
	if got := InferPrompt("hostname# "); got != "hostname#" && got != "#" {
		t.Fatalf("cisco prompt: %q", got)
	}
	if got := InferPrompt("user@host:~$ "); got == "" {
		t.Fatal("expected shell prompt")
	}
	if got := InferPrompt("no prompt here"); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestRecorderNoteOutputWaitFor(t *testing.T) {
	r := NewRecorder("Rec")
	r.Note("show ver\n")
	r.NoteOutput("Cisco IOS\nhostname# ")
	s := r.Script()
	if len(s.Steps) != 1 || s.Steps[0].WaitFor == "" {
		t.Fatalf("expected wait_for, got %+v", s.Steps)
	}
	if s.Steps[0].TimeoutMs != 15000 {
		t.Fatalf("timeout %d", s.Steps[0].TimeoutMs)
	}
}

func TestRankNames(t *testing.T) {
	names := []string{"Show version", "Disable paging (Cisco)", "Juniper show chassis"}
	got := RankNames(names, "cisco switch down")
	if got[0] != "Disable paging (Cisco)" {
		t.Fatalf("expected Cisco first, got %v", got)
	}
}
