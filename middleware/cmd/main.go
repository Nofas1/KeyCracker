package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"io"
	"key_cracker/middleware"
	"key_cracker/middleware/config"
	"key_cracker/middleware/internal/repository"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type AuthRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

type AuthConnection struct {
	client *http.Client
	rep *repository.Repo
	logger *slog.Logger
}

func NewAuthConnection(client *http.Client, cfg *config.Config, logger *slog.Logger) *AuthConnection {
	rep, err := repository.NewRepository(logger)
    if err != nil {
        logger.Error(
			"authConnection failed to initialize repository",
			"error", err,
		)
        panic("failed to connect to database")
    }
    logger.Info("authConnection initialized successfully")
	return &AuthConnection{
		client: client,
		rep: rep,
		logger: logger,
	}
}

func LoginHandler(auth mw.AAA) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		token, err := auth.Login(req.Name, req.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		json.NewEncoder(w).Encode(AuthResponse{
			Token: token,
		})
	}
}

func RegisterHandler(auth mw.AAA) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AuthRequest
		body, _ := io.ReadAll(r.Body)
		fmt.Printf("%s", string(body))
		err := json.Unmarshal(body, &req)
		// err := json.NewDecoder(body).Decode(&req)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = auth.Register(req.Name, req.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func (a *AuthConnection) AuthHandler(auth mw.AAA, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		bearer_token := strings.Split(authHeader, " ")
		if strings.ToLower(bearer_token[0]) != "bearer" || len(bearer_token) != 2 {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		tokenString := bearer_token[1]

		if _, err := auth.Verify(tokenString); err != nil {
			a.logger.Error(
				"Token verification failed",
				"error", err,
			)
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func main() {
	var cfg_path string
	flag.StringVar(&cfg_path, "config", "config.yaml", "config path")
	flag.Parse()

	client := &http.Client{}
	cfg, err := config.LoadConfig(cfg_path)
	if err != nil {
		panic(err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	authConnection := NewAuthConnection(client, cfg, logger)

	authService, err := mw.New(time.Hour, logger, authConnection.rep)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/login", LoginHandler(authService))
	http.HandleFunc("/register", RegisterHandler(authService))
	http.Handle("/bot", authConnection.AuthHandler(authService))

	logger.Info("Listening on", "Adr", cfg.Gateway.Address, "Port", cfg.Gateway.Port)
	err = http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.Gateway.Address, cfg.Gateway.Port), nil)
	fmt.Println(err)
}