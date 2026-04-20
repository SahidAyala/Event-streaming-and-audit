package auth

import "context"

type contextKey struct{}

// Identity carries the authenticated subject's authorisation data.
// It is injected into the request context by the HTTP auth middleware and
// consumed by application services to enforce tenant isolation.
type Identity struct {
	SubjectID string
	TenantID  string
	Roles     []string
}

// WithIdentity returns a copy of ctx with the Identity stored as a value.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, contextKey{}, id)
}

// IdentityFromContext retrieves the Identity stored by WithIdentity.
// Returns the zero value and false if no identity is present.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	v, ok := ctx.Value(contextKey{}).(Identity)
	return v, ok
}
