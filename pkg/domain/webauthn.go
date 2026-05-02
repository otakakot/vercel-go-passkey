package domain

import (
	"cmp"
	"net/http"
	"strings"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

var webAuthn *webauthn.WebAuthn

func NewWebAuthn(
	req *http.Request,
) (*webauthn.WebAuthn, error) {
	if webAuthn != nil {
		return webAuthn, nil
	}

	schema := cmp.Or(req.URL.Scheme, "https")

	rpid := strings.Split(req.Host, ":")[0]

	if rpid == "localhost" {
		schema = "http"
	}

	rpOrigin := schema + "://" + req.Host

	wa, err := webauthn.New(&webauthn.Config{
		RPID:          rpid,
		RPDisplayName: "otakakot-webauthn",
		RPOrigins:     []string{rpOrigin},
	})
	if err != nil {
		return nil, err
	}

	return wa, nil
}

var _ webauthn.User = (*User)(nil)

type User struct {
	ID          uuid.UUID
	Credentials []webauthn.Credential
}

func GenereteUser() *User {
	return &User{
		ID: uuid.New(),
	}
}

// WebAuthnCredentials implements webauthn.User.
func (user *User) WebAuthnCredentials() []webauthn.Credential {
	return user.Credentials
}

// WebAuthnDisplayName implements webauthn.User.
func (user *User) WebAuthnDisplayName() string {
	return user.ID.String()
}

// WebAuthnID implements webauthn.User.
func (user *User) WebAuthnID() []byte {
	return []byte(user.ID.String())
}

// WebAuthnName implements webauthn.User.
func (user *User) WebAuthnName() string {
	return user.ID.String()
}
