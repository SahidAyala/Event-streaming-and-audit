package jwt

import (
	"context"
	"net/http"
	"testing"
	"time"

	gjwt "github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-hmac-secret"

// makeToken signs a token with the test secret and returns the raw string.
func makeToken(c claims) string {
	token := gjwt.NewWithClaims(gjwt.SigningMethodHS256, c)
	s, _ := token.SignedString([]byte(testSecret))
	return s
}

func requestWithBearer(token string) *http.Request {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	return r
}

func TestJWTAuth_ValidTokenGrantsAccess(t *testing.T) {
	auth := New(testSecret)
	token := makeToken(claims{
		RegisteredClaims: gjwt.RegisteredClaims{
			Subject:   "user-123",
			ExpiresAt: gjwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		TenantID: "acme-corp",
		Roles:    []string{"writer", "reader"},
	})

	identity, err := auth.Authenticate(context.Background(), requestWithBearer(token))

	if err != nil {
		t.Fatalf("valid JWT must succeed, got: %v", err)
	}
	if identity.SubjectID != "user-123" {
		t.Errorf("SubjectID = %q, want %q", identity.SubjectID, "user-123")
	}
	if identity.TenantID != "acme-corp" {
		t.Errorf("TenantID = %q, want %q", identity.TenantID, "acme-corp")
	}
	if len(identity.Roles) != 2 {
		t.Errorf("Roles length = %d, want 2", len(identity.Roles))
	}
}

func TestJWTAuth_WrongSecretDeniesAccess(t *testing.T) {
	auth := New(testSecret)
	// Sign with a different secret.
	token := makeToken(claims{
		RegisteredClaims: gjwt.RegisteredClaims{Subject: "u", ExpiresAt: gjwt.NewNumericDate(time.Now().Add(time.Hour))},
		TenantID:         "tenant-a",
	})
	// Tamper: re-sign with different key — just use a different auth instance to verify.
	authOther := New("other-secret")
	badToken, _ := gjwt.NewWithClaims(gjwt.SigningMethodHS256, claims{
		RegisteredClaims: gjwt.RegisteredClaims{Subject: "u", ExpiresAt: gjwt.NewNumericDate(time.Now().Add(time.Hour))},
		TenantID:         "tenant-a",
	}).SignedString([]byte("other-secret"))
	_ = authOther
	_ = token

	_, err := auth.Authenticate(context.Background(), requestWithBearer(badToken))

	if err == nil {
		t.Fatal("token signed with wrong secret must be rejected")
	}
}

func TestJWTAuth_ExpiredTokenDeniesAccess(t *testing.T) {
	auth := New(testSecret)
	token := makeToken(claims{
		RegisteredClaims: gjwt.RegisteredClaims{
			Subject:   "user-123",
			ExpiresAt: gjwt.NewNumericDate(time.Now().Add(-time.Hour)), // expired
		},
		TenantID: "acme-corp",
	})

	_, err := auth.Authenticate(context.Background(), requestWithBearer(token))

	if err == nil {
		t.Fatal("expired token must be rejected")
	}
}

func TestJWTAuth_MissingTenantIDDeniesAccess(t *testing.T) {
	auth := New(testSecret)
	// Token without tenant_id claim.
	token := makeToken(claims{
		RegisteredClaims: gjwt.RegisteredClaims{
			Subject:   "user-123",
			ExpiresAt: gjwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		TenantID: "", // missing
	})

	_, err := auth.Authenticate(context.Background(), requestWithBearer(token))

	if err == nil {
		t.Fatal("token missing tenant_id must be rejected")
	}
}

func TestJWTAuth_MissingAuthorizationHeader(t *testing.T) {
	auth := New(testSecret)
	r, _ := http.NewRequest(http.MethodGet, "/", nil)

	_, err := auth.Authenticate(context.Background(), r)

	if err == nil {
		t.Fatal("missing Authorization header must return an error")
	}
}

func TestJWTAuth_MalformedBearerFormat(t *testing.T) {
	auth := New(testSecret)
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "notbearer tokenvalue")

	_, err := auth.Authenticate(context.Background(), r)

	if err == nil {
		t.Fatal("non-Bearer authorization must be rejected")
	}
}
