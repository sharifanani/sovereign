package auth

import (
	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/sovereign-im/sovereign/server/internal/store"
)

// webauthnUser adapts our User/Credential models to the go-webauthn User interface.
type webauthnUser struct {
	id          []byte
	name        string
	displayName string
	credentials []webauthn.Credential
}

func (u *webauthnUser) WebAuthnID() []byte {
	return u.id
}

func (u *webauthnUser) WebAuthnName() string {
	return u.name
}

func (u *webauthnUser) WebAuthnDisplayName() string {
	return u.displayName
}

func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

// newWebAuthnUser creates a webauthnUser from a store.User and its credentials.
func newWebAuthnUser(user *store.User, creds []*store.Credential) *webauthnUser {
	waCreds := make([]webauthn.Credential, len(creds))
	for i, c := range creds {
		waCreds[i] = storeCredToWebAuthn(c)
	}
	return &webauthnUser{
		id:          []byte(user.ID),
		name:        user.Username,
		displayName: user.DisplayName,
		credentials: waCreds,
	}
}

// storeCredToWebAuthn converts a store.Credential to a webauthn.Credential.
func storeCredToWebAuthn(c *store.Credential) webauthn.Credential {
	return webauthn.Credential{
		ID:        c.CredentialID,
		PublicKey: c.PublicKey,
		Authenticator: webauthn.Authenticator{
			SignCount: uint32(c.SignCount),
		},
	}
}
