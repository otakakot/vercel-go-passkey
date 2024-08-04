package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"github.com/otakakot/vercel-go-passkey/pkg/domain"
	"github.com/otakakot/vercel-go-passkey/pkg/kv"
	"github.com/otakakot/vercel-go-passkey/pkg/postgres"
	"github.com/otakakot/vercel-go-passkey/pkg/schema"
)

func Handler(
	rw http.ResponseWriter,
	req *http.Request,
) {
	switch req.Method {
	case http.MethodGet:
		Get(rw, req)

		return
	case http.MethodPost:
		Post(rw, req)

		return
	}

	http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
}

const cookey = "__attestation__"

func Get(
	rw http.ResponseWriter,
	req *http.Request,
) {
	slog.InfoContext(req.Context(), "attestation get begin")
	defer slog.InfoContext(req.Context(), "attestation get end")

	wa, err := domain.NewWebAuthn(req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	user := domain.GenereteUser()

	creation, session, err := wa.BeginRegistration(user)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	kv, err := kv.New[webauthn.SessionData](req.Context(), os.Getenv("KV_URL"))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	sid := uuid.New().String()

	if err := kv.Set(req.Context(), sid, *session); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	http.SetCookie(rw, &http.Cookie{
		Name:  cookey,
		Value: sid,
	})

	res := bytes.Buffer{}

	if err := json.NewEncoder(&res).Encode(creation.Response); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	rw.Header().Set("Content-Type", "application/json")

	rw.Write(res.Bytes())
}

func Post(
	rw http.ResponseWriter,
	req *http.Request,
) {
	slog.InfoContext(req.Context(), "attestation post begin")
	defer slog.InfoContext(req.Context(), "attestation post end")

	wa, err := domain.NewWebAuthn(req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	kv, err := kv.New[webauthn.SessionData](req.Context(), os.Getenv("KV_URL"))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	sid, err := req.Cookie(cookey)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	session, err := kv.GetDel(req.Context(), sid.Value)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	user := &domain.User{
		ID: uuid.MustParse(string(session.UserID)),
	}

	credential, err := wa.FinishRegistration(user, session, req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	pool, err := postgres.NewPool(req.Context(), os.Getenv("POSTGRES_URL"))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	defer pool.Close()

	tx, err := pool.Begin(req.Context())
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	if _, err := schema.New(tx).InsertUser(req.Context(), user.ID); err != nil {
		if err := tx.Rollback(req.Context()); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)

			return
		}

		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	cre := bytes.Buffer{}

	if err := json.NewEncoder(&cre).Encode(credential); err != nil {
		if err := tx.Rollback(req.Context()); err != nil {

			return
		}

		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	if _, err := schema.New(tx).InsertWebAuthnCredential(req.Context(), schema.InsertWebAuthnCredentialParams{
		RawID:      credential.ID,
		UserID:     user.ID,
		Credential: cre.Bytes(),
	}); err != nil {
		if err := tx.Rollback(req.Context()); err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)

			return
		}

		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	if err := tx.Commit(req.Context()); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	http.SetCookie(rw, &http.Cookie{
		Name:   cookey,
		Value:  "",
		MaxAge: -1,
	})

	rw.WriteHeader(http.StatusCreated)
}
