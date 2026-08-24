package service

import (
	"strings"

	"github.com/lumen/relay/internal/auth"
	"github.com/lumen/relay/internal/config"
	"github.com/lumen/relay/internal/domain"
	"github.com/lumen/relay/internal/repo"
)

type Auth struct {
	Store repo.Store
	Cfg   config.Config
}

func (a Auth) Login(email, password string) (auth.Tokens, domain.Claims, error) {
	u, err := a.Store.UserByEmail(strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return auth.Tokens{}, domain.Claims{}, domain.ErrUnauthorized
	}
	if !auth.CheckPassword(u.PasswordHash, password) {
		return auth.Tokens{}, domain.Claims{}, domain.ErrUnauthorized
	}
	c := domain.Claims{UserID: u.ID, TenantID: u.TenantID, Email: u.Email, Role: u.Role}
	tok, err := auth.Issue(a.Cfg.JWTSecret, c, a.Cfg.JWTAccessMin, a.Cfg.JWTRefreshH)
	return tok, c, err
}

func (a Auth) Refresh(refresh string) (auth.Tokens, domain.Claims, error) {
	c, kind, err := auth.Parse(a.Cfg.JWTSecret, refresh)
	if err != nil || kind != "refresh" {
		return auth.Tokens{}, domain.Claims{}, domain.ErrUnauthorized
	}
	tok, err := auth.Issue(a.Cfg.JWTSecret, c, a.Cfg.JWTAccessMin, a.Cfg.JWTRefreshH)
	return tok, c, err
}
