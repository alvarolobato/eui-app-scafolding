package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/MicahParks/keyfunc"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"github.com/golang-jwt/jwt/v4"
	"github.com/julienschmidt/httprouter"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

const (
	googleStateCookieKey = "google_state"
)

var (
	// errUnauthorized is returned when the user has not yet authorized access.
	errUnauthorized = errors.New("unauthorized")

	// errInvalidState is returned when OAuth state validation fails.
	errInvalidState = errors.New("state does not match")
)

// authDetails holds information about an authenticated user.
type authDetails struct {
	idToken *jwt.Token
	claims  jwt.MapClaims
	userID  string
	name    string
	email   string
	picture string
}

type authKey struct{}

func authFromContext(ctx context.Context) *authDetails {
	return ctx.Value(authKey{}).(*authDetails)
}

// oauthStateData represents the data encoded in the OAuth state parameter.
type oauthStateData struct {
	Nonce []byte            `json:"nonce"`
	Data  map[string]string `json:"data,omitempty"`
}

// generateOAuthState generates a random "state" value for an OAuth 2.0 session,
// for protecting against CSRF attacks on the redirect handler.
func generateOAuthState(
	secureCookies secureCookies,
	cookieName, cookiePath string,
	additionalData map[string]string,
) (string, *http.Cookie, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return "", nil, fmt.Errorf("failed to generate state nonce: %w", err)
	}

	stateJSON, err := json.Marshal(oauthStateData{
		Nonce: nonce,
		Data:  additionalData,
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal state data: %w", err)
	}
	state := base64.URLEncoding.EncodeToString(stateJSON)

	cookieValue, err := secureCookies.Encode(state)
	if err != nil {
		return "", nil, fmt.Errorf("failed to encode state nonce: %w", err)
	}
	return state, &http.Cookie{
		Name:     cookieName,
		Path:     cookiePath,
		Value:    cookieValue,
		Secure:   true,
		HttpOnly: true,
	}, nil
}

// validateOAuthState validates the "state" query parameter matches the value
// in the cookie with the given name.
func validateOAuthState(
	secureCookies secureCookies,
	r *http.Request, cookieName string,
) (map[string]string, error) {
	state := r.URL.Query().Get("state")
	stateCookie, err := r.Cookie(cookieName)
	if err != nil {
		return nil, err
	}

	expected, err := secureCookies.Decode(stateCookie.Value)
	if err != nil {
		return nil, err
	}
	if subtle.ConstantTimeCompare([]byte(state), []byte(expected)) != 1 {
		return nil, errInvalidState
	}

	// Decode state to get the additional data
	stateDecoded, err := base64.URLEncoding.DecodeString(state)
	if err != nil {
		return nil, fmt.Errorf("failed to decode state: %w", err)
	}
	var stateData oauthStateData
	if err := json.Unmarshal(stateDecoded, &stateData); err == nil {
		return stateData.Data, nil
	}

	return nil, nil
}

// idTokenParser creates a function that parses and validates Google ID tokens.
func idTokenParser(jwks *keyfunc.JWKS, googleClientID string) func(string) (*authDetails, error) {
	return func(idToken string) (*authDetails, error) {
		token, err := jwt.Parse(idToken, jwks.Keyfunc, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Name}))
		if err != nil {
			return nil, err
		}
		claims := token.Claims.(jwt.MapClaims)
		if !claims.VerifyAudience(googleClientID, true) {
			return nil, errors.New("audience invalid or missing")
		}

		picture, _ := claims["picture"].(string)
		name, _ := claims["name"].(string)
		return &authDetails{
			idToken: token,
			claims:  claims,
			userID:  claims["sub"].(string),
			email:   claims["email"].(string),
			name:    name,
			picture: picture,
		}, nil
	}
}

// oauth2ConfigForURL returns a copy of given oauth2.Config with the redirect
// URL made absolute using the request headers.
func oauth2ConfigForURL(cfg oauth2.Config, r *http.Request) *oauth2.Config {
	url := url.URL{Scheme: "http", Host: r.Host, Path: cfg.RedirectURL}
	if xfh := r.Header.Get("X-Forwarded-Host"); xfh != "" {
		url.Host = xfh
	}
	if xfp := r.Header.Get("X-Forwarded-Proto"); xfp != "" {
		url.Scheme = xfp
	} else if r.TLS != nil {
		url.Scheme = "https"
	}
	cfg.RedirectURL = url.String()
	return &cfg
}

// getAuthMiddleware creates middleware that validates authentication.
func getAuthMiddleware(
	secureCookies secureCookies,
	parseIDToken func(string) (*authDetails, error),
) func(h httprouter.Handle) httprouter.Handle {
	return func(h httprouter.Handle) httprouter.Handle {
		return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
			cookie, err := r.Cookie("credentials")
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			credentials, err := secureCookies.Decode(cookie.Value)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			details, err := parseIDToken(credentials)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			if span := trace.SpanFromContext(r.Context()); span != nil {
				span.SetAttributes(
					attribute.String("user.id", details.userID),
					attribute.String("user.email", details.email),
				)
			}
			r = r.WithContext(context.WithValue(r.Context(), authKey{}, details))
			h(w, r, p)
		}
	}
}

// tokenStorage manages OAuth tokens for Google.
type tokenStorage struct {
	googleConfig oauth2.Config
	client       *elasticsearch.Client
	logger       *zap.Logger

	mu           sync.RWMutex
	googleTokens map[string]*oauth2.Token
}

// tokenDocument represents a token document in Elasticsearch.
type tokenDocument struct {
	Google struct {
		RefreshToken string `json:"refresh_token"`
	} `json:"google"`
}

// newTokenStorage creates a new tokenStorage instance.
func newTokenStorage(
	googleConfig oauth2.Config,
	client *elasticsearch.Client, logger *zap.Logger,
) (*tokenStorage, error) {
	s := &tokenStorage{
		googleConfig: googleConfig,
		googleTokens: make(map[string]*oauth2.Token),
		client:       client,
		logger:       logger,
	}
	if err := s.init(logger); err != nil {
		return nil, fmt.Errorf("failed to init token storage: %w", err)
	}
	return s, nil
}

// init loads existing tokens from Elasticsearch.
func (s *tokenStorage) init(logger *zap.Logger) error {
	if s.client == nil {
		return nil
	}

	ctx, span := otel.Tracer("main").Start(context.Background(), "initTokenStorage")
	defer span.End()
	logger = logger.With(traceLogFields(ctx)...)

	// Search for all token documents
	res, err := s.client.Search(
		s.client.Search.WithContext(ctx),
		s.client.Search.WithIndex("app-sessions"),
		s.client.Search.WithSize(1000),
	)
	if err != nil {
		// Index might not exist yet
		logger.Info("could not load tokens from Elasticsearch", zap.Error(err))
		return nil
	}
	defer res.Body.Close()

	if res.IsError() {
		logger.Info("could not load tokens from Elasticsearch", zap.String("status", res.Status()))
		return nil
	}

	var searchResult struct {
		Hits struct {
			Hits []struct {
				ID     string        `json:"_id"`
				Source tokenDocument `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	for _, hit := range searchResult.Hits.Hits {
		if hit.Source.Google.RefreshToken != "" {
			s.googleTokens[hit.ID] = &oauth2.Token{
				TokenType:    "Bearer",
				RefreshToken: hit.Source.Google.RefreshToken,
			}
		}
	}

	logger.Info(
		"loaded OAuth tokens",
		zap.Int("google_tokens", len(s.googleTokens)),
	)

	span.SetStatus(codes.Ok, "")
	return nil
}

// setGoogle sets a Google OAuth token for a user.
func (s *tokenStorage) setGoogle(ctx context.Context, id string, token *oauth2.Token) error {
	s.mu.Lock()
	s.googleTokens[id] = token
	s.mu.Unlock()

	if s.client != nil {
		return s.putToken(ctx, "google", id, token)
	}
	return nil
}

// putToken persists an OAuth token to Elasticsearch.
func (s *tokenStorage) putToken(ctx context.Context, typ, id string, token *oauth2.Token) error {
	if token.RefreshToken == "" {
		return fmt.Errorf("empty refresh token for user ID %q", id)
	}

	body := esutil.NewJSONReader(map[string]interface{}{
		"doc_as_upsert": true,
		"doc": map[string]interface{}{
			typ: map[string]interface{}{
				"issued_at":     time.Now().UTC().Format(time.RFC3339),
				"refresh_token": token.RefreshToken,
			},
		},
	})
	res, err := s.client.Update("app-sessions", id, body, s.client.Update.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("while saving token for user ID %q: %w", id, err)
	}
	if res.IsError() {
		defer res.Body.Close()
		return fmt.Errorf("updating token failed: %s", res.Status())
	}
	defer res.Body.Close()
	return nil
}

// getGoogle gets a Google OAuth token for a user, refreshing it if necessary.
func (s *tokenStorage) getGoogle(ctx context.Context, id string, r *http.Request) (*oauth2.Token, error) {
	s.mu.RLock()
	token := s.googleTokens[id]
	s.mu.RUnlock()
	if token == nil || token.RefreshToken == "" {
		return nil, errUnauthorized
	}

	newToken, err := oauth2ConfigForURL(s.googleConfig, r).TokenSource(ctx, token).Token()
	if err != nil {
		return nil, err
	}

	if token.AccessToken != newToken.AccessToken {
		s.logger.Info("refreshed google token", zap.String("id", id))
		s.mu.Lock()
		s.googleTokens[id] = newToken
		s.mu.Unlock()
	}
	if token.RefreshToken != newToken.RefreshToken {
		if err := s.setGoogle(ctx, id, newToken); err != nil {
			return nil, err
		}
	}
	return newToken, nil
}

// newGoogleOAuthConfig creates a Google OAuth2 configuration.
func newGoogleOAuthConfig(clientID, clientSecret string) oauth2.Config {
	return oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     endpoints.Google,
		RedirectURL:  "/api/oauth/google",
		Scopes:       []string{"openid", "email", "profile"},
	}
}
