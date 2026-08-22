package itglue

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientListOrganizations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/organizations" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":   "123",
					"type": "organizations",
					"attributes": map[string]string{"name": "Contoso"},
				},
			},
			"meta": map[string]int{"current-page": 1, "next-page": 0, "total-pages": 1},
		})
	}))
	defer srv.Close()

	c := New("test-key", srv.URL)
	orgs, err := c.ListOrganizations(context.Background())
	if err != nil || len(orgs) != 1 || orgs[0].Name != "Contoso" {
		t.Fatalf("%+v %v", orgs, err)
	}
}

func TestGetPasswordShow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/passwords/99" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":   "99",
				"type": "passwords",
				"attributes": map[string]string{
					"name":     "Router",
					"username": "admin",
					"password": "secret-value",
				},
			},
		})
	}))
	defer srv.Close()

	c := New("key", srv.URL)
	p, err := c.GetPassword(context.Background(), "99")
	if err != nil || p.Password != "secret-value" || p.Username != "admin" {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestVaultCredentialName(t *testing.T) {
	n := VaultCredentialName(Password{OrganizationName: "Contoso", Name: "Core switch"})
	if n == "" || len(n) > 80 {
		t.Fatal(n)
	}
}
