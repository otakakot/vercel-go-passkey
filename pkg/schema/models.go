// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package schema

import (
	"github.com/google/uuid"
)

type User struct {
	ID uuid.UUID `json:"id"`
}

type WebauthnCredential struct {
	RawID      []byte    `json:"raw_id"`
	UserID     uuid.UUID `json:"user_id"`
	Credential []byte    `json:"credential"`
}
