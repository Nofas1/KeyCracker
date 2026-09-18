package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"

	"io"
	"key_cracker/middleware"
	"key_cracker/middleware/config"
	"key_cracker/middleware/internal/repository"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	// "google.golang.org/api/calendar/v3"
	// "google.golang.org/api/option"
)

type AuthRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
}

type AuthConnection struct {
	rep    *repository.Repo
	logger *slog.Logger
}

func NewAuthConnection(cfg *config.Config, logger *slog.Logger) *AuthConnection {
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
		rep:    rep,
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

func VerifyHandler(auth mw.AAA) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read token", http.StatusBadRequest)
			return
		}

		user, err := auth.Verify(string(tokenString))
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"user":   user,
			"status": "valid",
		})
	}
}

func (a *AuthConnection) AuthHandler(auth mw.AAA, next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || strings.ToLower(bearerToken[0]) != "bearer" {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		tokenString := bearerToken[1]

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

// getTokenFromWeb запрашивает OAuth-код у пользователя и обменивает его на токен.
func getTokenFromWeb(cfg *oauth2.Config) (*oauth2.Token, error) {
	authURL := cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		return nil, fmt.Errorf("unable to read authorization code: %w", err)
	}

	tok, err := cfg.Exchange(context.TODO(), authCode)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}
	return tok, nil
}

func (a *AuthConnection) GoogleAuthHandler(auth mw.AAA, name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := a.rep.GetToken(r.Context(), name)
		if errors.Is(err, repository.UserNotExist) {
			oauthCfg, cfgErr := google.ConfigFromJSON([]byte{}, "https://www.googleapis.com/auth/userinfo.email")
			if cfgErr != nil {
				a.logger.Error("failed to load google oauth config", "error", cfgErr)
				http.Error(w, "OAuth config error", http.StatusInternalServerError)
				return
			}
			tok, tokErr := getTokenFromWeb(oauthCfg)
			if tokErr != nil {
				a.logger.Error("failed to get token from web", "error", tokErr)
				http.Error(w, "OAuth flow failed", http.StatusInternalServerError)
				return
			}
			// TODO: сохранить tok в репозиторий
			_ = tok
			return
		}
		if err != nil {
			a.logger.Error("failed to get token", "error", err)
			http.Error(w, "", http.StatusForbidden)
			return
		}

		_ = token
		// TODO: использовать токен для запроса к Google API
	}
}

func main() {
	var cfgPath string
	flag.StringVar(&cfgPath, "config", "config.yaml", "config path")
	flag.Parse()

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		panic(err)
	}

	if cfg.SecretKey == "" {
		panic("SECRET_KEY must be set in config")
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	authConnection := NewAuthConnection(cfg, logger)

	authService, err := mw.New(time.Hour, logger, authConnection.rep, cfg.SecretKey)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/login", LoginHandler(authService))
	http.HandleFunc("/register", RegisterHandler(authService))
	http.HandleFunc("/verify", VerifyHandler(authService))

	logger.Info("Listening on", "Adr", cfg.Gateway.Address, "Port", cfg.Gateway.Port)
	err = http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.Gateway.Address, cfg.Gateway.Port), nil)
	fmt.Println(err)
}