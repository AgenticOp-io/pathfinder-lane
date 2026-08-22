package auvik

import (
	"encoding/json"
	"testing"
)

func TestSessionNodesFilters(t *testing.T) {
	devs := []Device{
		{ID: "1", Name: "core-sw", IPs: []string{"10.0.0.1"}, DeviceType: "switch", LoginStatus: "authorized"},
		{ID: "2", Name: "pc", IPs: []string{"10.0.0.2"}, DeviceType: "workstation", LoginStatus: "authorized"},
		{ID: "3", Name: "noip", DeviceType: "router", LoginStatus: "authorized"},
	}
	opts := ImportOptions{NetworkGearOnly: true, RequireLoginAuthorized: true}
	nodes, st := SessionNodes(devs, opts)
	if len(nodes) != 1 || nodes[0].Host != "10.0.0.1" {
		t.Fatalf("nodes %+v", nodes)
	}
	if st.Imported != 1 || st.Skipped != 1 || st.NoIP != 1 {
		t.Fatalf("stats %+v", st)
	}
}

func TestDecodeDeviceIncluded(t *testing.T) {
	raw := `{
	  "data": [{
	    "id": "dev1",
	    "attributes": {"ipAddresses":["192.168.1.1"],"deviceName":"r1","deviceType":"router"},
	    "relationships": {
	      "tenant": {"data": {"id": "t1"}},
	      "deviceDiscoveryStatus": {"data": {"id": "ds1"}}
	    }
	  }],
	  "included": [{
	    "type": "deviceDiscoveryStatus",
	    "id": "ds1",
	    "attributes": {"login": "authorized"}
	  }]
	}`
	var page devicesEnvelope
	if err := json.Unmarshal([]byte(raw), &page); err != nil {
		t.Fatal(err)
	}
	d := decodeDevice(page.Data[0], page.Included)
	if d.LoginStatus != "authorized" || d.PrimaryIP() != "192.168.1.1" {
		t.Fatalf("%+v", d)
	}
}
