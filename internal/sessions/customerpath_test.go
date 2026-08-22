package sessions

import "testing"

func TestCustomerOfFolder(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"Unassigned", ""},
		{"Customers", ""},
		{"Customers/Acme", "Acme"},
		{"Customers/Acme/Core", "Acme"},
		{"Customers/Acme Fiber/Site A", "Acme Fiber"},
	}
	for _, c := range cases {
		if got := CustomerOfFolder(c.in); got != c.want {
			t.Fatalf("CustomerOfFolder(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestCustomerTag(t *testing.T) {
	if got := CustomerTag(" Acme "); got != "customer:Acme" {
		t.Fatalf("got %q", got)
	}
}
