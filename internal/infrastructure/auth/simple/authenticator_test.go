package simple

import (
	"context"
	"net/http"
	"testing"
)

func TestSimpleAuth_ValidKeyGrantsAccess(t *testing.T) {
	auth := New("secret-key")
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-API-Key", "secret-key")

	identity, err := auth.Authenticate(context.Background(), r)

	if err != nil {
		t.Fatalf("valid API key must succeed, got: %v", err)
	}
	if identity.SubjectID == "" {
		t.Error("SubjectID must be set")
	}
	if identity.TenantID != "default" {
		t.Errorf("TenantID = %q, want %q", identity.TenantID, "default")
	}
	if len(identity.Roles) == 0 {
		t.Error("Roles must be non-empty")
	}
}

func TestSimpleAuth_WrongKeyDeniesAccess(t *testing.T) {
	auth := New("secret-key")
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-API-Key", "wrong-key")

	_, err := auth.Authenticate(context.Background(), r)

	if err == nil {
		t.Fatal("wrong API key must return an error")
	}
}

func TestSimpleAuth_MissingHeaderDeniesAccess(t *testing.T) {
	auth := New("secret-key")
	r, _ := http.NewRequest(http.MethodGet, "/", nil)

	_, err := auth.Authenticate(context.Background(), r)

	if err == nil {
		t.Fatal("missing X-API-Key header must return an error")
	}
}
