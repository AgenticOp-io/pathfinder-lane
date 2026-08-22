package crawlrun

import "testing"

func TestConfidenceReachedKnown(t *testing.T) {
	d := DeviceRow{State: StateReached, Platform: "cisco_ios", Neighbors: 3}
	if c := d.Confidence(); c < 80 {
		t.Fatalf("confidence=%d", c)
	}
}

func TestConfidenceFailed(t *testing.T) {
	d := DeviceRow{State: StateFailed, Platform: "unknown"}
	if c := d.Confidence(); c > 30 {
		t.Fatalf("confidence=%d", c)
	}
}
