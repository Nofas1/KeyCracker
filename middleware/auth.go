package mw

import (
	"context"
	"log/slog"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"key_cracker/middleware/internal/repository"
)

const adminRole = "superuser" // token subject

// Authentication, Authorization, Accounting
type AAA struct {
	rep       *repository.Repo
	tokenTTL  time.Duration
	logger       *slog.Logger
	secretKey string
}

func New(tokenTTL time.Duration, logger *slog.Logger, rep *repository.Repo, secretKey string) (AAA, error) {
	const adminUser = "ADMIN_USER"
	const adminPass = "ADMIN_PASSWORD"
	user, ok := os.LookupEnv(adminUser)
	if !ok {
		return AAA{}, fmt.Errorf("could not get admin user from environment")
	}
	password, ok := os.LookupEnv(adminPass)
	if !ok {
		return AAA{}, fmt.Errorf("could not get admin password from environment")
	}

	// Register admin user on startup if not exists
	if err := rep.RegisterUser(context.Background(), user, password); err != nil {
		logger.Warn("admin user may already exist", "error", err)
	}

	return AAA{
		rep:       rep,
		tokenTTL:  tokenTTL,
		logger:    logger,
		secretKey: secretKey,
	}, nil
}

func (a *AAA) Register(name, password string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	if err := a.rep.RegisterUser(context.Background(), name, string(hashed)); err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	}
	// if _, exists := a.users[name]; !exists {
	// 	a.users[name] = password
	// } else {
	// 	return errors.New("User already exists")
	// }
	return nil
}

func (a *AAA) Login(name, password string) (string, error) {
	hashed, err := a.rep.GetPasswordHash(context.Background(), name)
	if err != nil {
		return "", errors.New("invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password)); err != nil {
		return "", errors.New("invalid credentials")
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   name,
		Issuer:    "key_cracker_middleware",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.tokenTTL)),
	})
	token, err := jwtToken.SignedString([]byte(a.secretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return token, nil
}

func (a *AAA) Verify(tokenString string) (string, error) {
	claims := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (any, error) {
		// Validate signing method
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(a.secretKey), nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return "", errors.New("token expired or invalid")
	}

	// Validate issuer
	if claims.Issuer != "key_cracker_middleware" {
		return "", errors.New("invalid token issuer")
	}

	return claims.Subject, nil
}