package utils

import (
	"errors"
	"fmt"
	"net/http"
	"time"
	"video-processing/models"

	"github.com/google/uuid"
	"github.com/o1egl/paseto"
)

var (
	ErrInvalidSigningKey = errors.New("invalid signing key")
	ErrInvalidToken      = errors.New("invalid token")
)

type Payload struct {
	ID       uuid.UUID `json:"id"`
	IssuedAt time.Time `json:"issued_at"`
	ExpireAt time.Time `json:"expire_at"`
}

func (p Payload) valid() bool {
	return p.ExpireAt.After(time.Now())
}

func NewPayload(id uuid.UUID, duration time.Duration) Payload {
	return Payload{
		ID:       id,
		IssuedAt: time.Now(),
		ExpireAt: time.Now().Add(duration),
	}
}

type TokenManager interface {
	CreateToken(p Payload) (string, error)
	VerifyToken(token string) (Payload, error)
}
type tokenManager struct {
	key    string
	paseto paseto.V2
	dur    time.Duration
}

func NewTokenManager(key string, duration time.Duration, p paseto.V2) TokenManager {
	return &tokenManager{
		key:    key,
		paseto: p,
		dur:    duration,
	}
}

func (tm tokenManager) CreateToken(p Payload) (string, error) {
	p.ExpireAt = p.IssuedAt.Add(tm.dur)
	if len(tm.key) != 32 {
		return "", models.Error{
			Code:        http.StatusInternalServerError,
			Message:     "internal server error",
			Description: "failed to create token",
			Params:      fmt.Sprintf("payload:%v", p),
			Err:         fmt.Errorf("bad key length %d", len(tm.key)),
		}
	}
	token, err := tm.paseto.Encrypt([]byte(tm.key), p, nil)
	if err != nil {
		return "", models.Error{
			Code:        http.StatusInternalServerError,
			Message:     "internal server error",
			Description: "failed to create token",
			Params:      fmt.Sprintf("payload:%v", p),
			Err:         fmt.Errorf("failed to create token: %w", err),
		}
	}
	return token, nil
}

func (tm tokenManager) VerifyToken(token string) (Payload, error) {
	payload := &Payload{}

	err := tm.paseto.Decrypt(token, []byte(tm.key), payload, nil)
	if err != nil {
		return Payload{}, models.Error{
			Code:        http.StatusInternalServerError,
			Message:     "internal server error",
			Description: "failed to verify token",
			Params:      fmt.Sprintf("token:%v", token),
			Err:         fmt.Errorf("failed to verify token: %w", err),
		}
	}
	if !payload.valid() {
		return Payload{}, models.Error{
			Code:        http.StatusUnauthorized,
			Message:     "unauthorized",
			Description: "invalid access token",
			Params:      fmt.Sprintf("token:%v", token),
			Err:         fmt.Errorf("token expired"),
		}
	}

	return *payload, nil
}
