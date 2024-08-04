-- name: InsertUser :one
INSERT INTO
    users (id)
VALUES
    ($1) RETURNING *;

-- name: InsertWebAuthnCredential :one
INSERT INTO
    webauthn_credentials (raw_id, user_id, credential)
VALUES
    ($1, $2, $3) RETURNING *;

-- name: FindWebAuthnCredentialByUserID :one
SELECT
    *
FROM
    webauthn_credentials
WHERE
    user_id = $1;
