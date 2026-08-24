package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lumen/relay/internal/clock"
	"github.com/lumen/relay/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

func Issue(secret string, c domain.Claims, accessMin, refreshH int) (Tokens, error) {
	now := clock.Now()
	access, err := sign(secret, c, "access", now.Add(time.Duration(accessMin)*time.Minute))
	if err != nil {
		return Tokens{}, err
	}
	refresh, err := sign(secret, c, "refresh", now.Add(time.Duration(refreshH)*time.Hour))
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: refresh, ExpiresIn: accessMin * 60}, nil
}

func Parse(secret, raw string) (domain.Claims, string, error) {
	tok, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("alg")
		}
		return []byte(secret), nil
	})
	if err != nil || !tok.Valid {
		return domain.Claims{}, "", domain.ErrUnauthorized
	}
	mc, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return domain.Claims{}, "", domain.ErrUnauthorized
	}
	kind, _ := mc["typ"].(string)
	return domain.Claims{
		UserID:   str(mc["uid"]),
		TenantID: str(mc["tid"]),
		Email:    str(mc["eml"]),
		Role:     str(mc["role"]),
	}, kind, nil
}

func sign(secret string, c domain.Claims, typ string, exp time.Time) (string, error) {
	now := clock.Now()
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid":  c.UserID,
		"tid":  c.TenantID,
		"eml":  c.Email,
		"role": c.Role,
		"typ":  typ,
		"iat":  now.Unix(),
		"exp":  exp.Unix(),
	})
	return t.SignedString([]byte(secret))
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
