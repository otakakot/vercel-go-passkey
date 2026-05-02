package api

import (
	"bytes"
	"encoding/json"
	"fmt"
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

const cookey = "__assertion__"

func Get(
	rw http.ResponseWriter,
	req *http.Request,
) {
	slog.InfoContext(req.Context(), "assertion get begin")
	defer slog.InfoContext(req.Context(), "assertion get end")

	wa, err := domain.NewWebAuthn(req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	assertion, session, err := wa.BeginDiscoverableLogin()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	cache, err := kv.New[webauthn.SessionData](req.Context(), os.Getenv("KV_URL"))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	sid := uuid.New().String()

	if err := cache.Set(req.Context(), sid, *session); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	http.SetCookie(rw, &http.Cookie{
		Name:  cookey,
		Value: sid,
	})

	if err := json.NewEncoder(rw).Encode(assertion.Response); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	rw.Header().Set("Content-Type", "application/json")
}

func Post(
	rw http.ResponseWriter,
	req *http.Request,
) {
	slog.InfoContext(req.Context(), "assertion post begin")
	defer slog.InfoContext(req.Context(), "assertion post end")

	wa, err := domain.NewWebAuthn(req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	cache, err := kv.New[webauthn.SessionData](req.Context(), os.Getenv("KV_URL"))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	sid, err := req.Cookie(cookey)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	session, err := cache.GetDel(req.Context(), sid.Value)
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

	userHandler := func(rawID, userHandle []byte) (webauthn.User, error) {
		uid, err := uuid.Parse(string(userHandle))
		if err != nil {
			return nil, err
		}

		cre, err := schema.New(pool).FindWebAuthnCredentialByUserID(req.Context(), uid)
		if err != nil {
			return nil, err

		}

		if !bytes.Equal(rawID, cre.RawID) {
			return nil, fmt.Errorf("raw id mismatch")
		}

		credential := webauthn.Credential{}

		if err := json.NewDecoder(bytes.NewBuffer(cre.Credential)).Decode(&credential); err != nil {
			return nil, err
		}

		return &domain.User{
			ID:          uid,
			Credentials: []webauthn.Credential{credential},
		}, nil
	}

	if _, err := wa.FinishDiscoverableLogin(func(rawID, userHandle []byte) (user webauthn.User, err error) {
		return userHandler(rawID, userHandle)
	}, session, req); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)

		return
	}

	http.SetCookie(rw, &http.Cookie{
		Name:   cookey,
		Value:  "",
		MaxAge: -1,
	})
}
