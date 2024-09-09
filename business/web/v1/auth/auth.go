// Package auth provides authentication and authorization support.
// Authentication: You are who you say you are.
// Authorization:  You have permission to do what you are requesting to do.
package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/navigacontentlab/panurge/navigaid"
	"github.com/vikaskumar1187/publisher_saas/foundation/logger"
)

// Config represents information required to initialize auth.
type Config struct {
	Log         *logger.Logger
	Env         string
	ImasURL     string
	Permissions string
}

// Auth is used to authenticate clients.
type Auth struct {
	log         *logger.Logger
	method      jwt.SigningMethod
	parser      *jwt.Parser
	env         string
	imasURL     string
	permissions string
}

// New creates an Auth to support authentication/authorization.
func New(cfg Config) (*Auth, error) {

	a := Auth{
		log:         cfg.Log,
		method:      jwt.GetSigningMethod(jwt.SigningMethodRS256.Name),
		parser:      jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name})),
		env:         cfg.Env,
		imasURL:     cfg.ImasURL,
		permissions: cfg.Permissions,
	}
	return &a, nil
}

// Authenticate processes the token to validate the sender's token is valid.
func (a *Auth) Authenticate(ctx context.Context, bearerToken string) (navigaid.Claims, error) {
	parts := strings.Split(bearerToken, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return navigaid.Claims{}, errors.New("expected authorization header format: Bearer <token>")
	}

	jwks := navigaid.NewJWKS(
		navigaid.ImasJWKSEndpoint(a.imasURL),
	)

	var claims navigaid.Claims
	claims, err := jwks.Validate(bearerToken)
	if err != nil {
		return navigaid.Claims{}, fmt.Errorf("error parsing token: %w", err)
	}

	return claims, nil
}

// Check if the user has permissions in the organization
func (a *Auth) Authorize(ctx context.Context, claims navigaid.Claims) error {

	hasPerm := claims.HasPermissionsInOrganisation(a.permissions)
	if !hasPerm {
		return fmt.Errorf("not enough permissions")
	}

	return nil
}
