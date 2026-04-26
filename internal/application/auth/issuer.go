package auth

import "time"

// IssueCommand carries the parameters for minting a new token.
type IssueCommand struct {
	// Identity of the authenticated caller — subject, tenant_id, and roles are
	// embedded into the issued token so it carries the same authorization scope.
	Identity Identity
	// ExpiresIn is how long the token should be valid. Zero uses the default (1 hour).
	ExpiresIn time.Duration
}

// IssuedToken is the result of a successful token issuance.
type IssuedToken struct {
	Token     string
	ExpiresAt time.Time
}

// Issuer is the port for minting new access tokens.
// It is intentionally separate from Authenticator: an implementation may support
// validation but not issuance (e.g. a third-party OIDC provider).
type Issuer interface {
	Issue(cmd IssueCommand) (IssuedToken, error)
}
