package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/MicahParks/keyfunc"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/julienschmidt/httprouter"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	serviceName = "app-backend"
)

func main() {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "@timestamp"
	encoderConfig.LevelKey = "log.level"
	encoderConfig.NameKey = "log.logger"
	encoderConfig.FunctionKey = "code.function.name"
	encoderConfig.StacktraceKey = "code.stacktrace"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		zap.DebugLevel,
	)
	logger := zap.New(core, zap.AddCaller())
	zap.ReplaceGlobals(logger)

	shutdown, err := initOpenTelemetry(context.Background(), serviceName)
	if err != nil {
		logger.Fatal("failed to init OpenTelemetry", zap.Error(err))
	}
	defer shutdown(context.Background())

	configPath := flag.String("c", "", "path to configuration file")
	flag.Parse()

	config, err := loadConfig(*configPath)
	if err != nil {
		logger.Fatal("while loading config", zap.Error(err))
	}
	secureCookies, err := newSecureCookies(config.EncryptionKeys)
	if err != nil {
		logger.Fatal("failed to construct secure cookie codecs", zap.Error(err))
	}
	if len(secureCookies) == 0 {
		logger.Warn("encryption_keys configuration unspecified: cookies will not be signed or encrypted")
	}

	apmServerURL := os.Getenv("ELASTIC_APM_SERVER_URL")
	if apmServerURL == "" {
		apmServerURL = "http://localhost:8200"
	}

	var esClient *elasticsearch.Client
	if config.Elasticsearch.APIKey == "" {
		logger.Info("Elasticsearch API Key not set, using in-memory storage")
	} else {
		client, err := elasticsearch.NewClient(elasticsearch.Config{
			Addresses:       []string{config.Elasticsearch.URL},
			APIKey:          config.Elasticsearch.APIKey,
			Instrumentation: elasticsearch.NewOpenTelemetryInstrumentation(otel.GetTracerProvider(), false),
		})
		if err != nil {
			logger.Fatal("failed to create Elasticsearch client", zap.Error(err))
		}
		esClient = client
	}

	// Instrument all outgoing HTTP requests
	http.DefaultClient.Transport = otelhttp.NewTransport(http.DefaultTransport)

	// Initialize Google JWKs for token validation
	googleJWKS, err := keyfunc.Get(
		"https://www.googleapis.com/oauth2/v3/certs",
		keyfunc.Options{RefreshInterval: time.Hour},
	)
	if err != nil {
		logger.Fatal("failed to obtain Google JWKS", zap.Error(err))
	}
	parseIDToken := idTokenParser(googleJWKS, config.Google.ClientID)

	googleConfig := newGoogleOAuthConfig(config.Google.ClientID, config.Google.ClientSecret)

	tokens, err := newTokenStorage(googleConfig, esClient, logger)
	if err != nil {
		logger.Fatal("failed to create token storage", zap.Error(err))
	}

	// Generate sample data
	sampleData := generateSampleData()

	router := httprouter.New()

	// Public endpoint: returns frontend configuration
	router.GET("/api/config", wrapHandler(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		var result struct {
			APM struct {
				ServerURL string `json:"server_url"`
			} `json:"apm"`

			Google struct {
				ClientID   string `json:"client_id"`
				OAuthScope string `json:"oauth_scope"`
			} `json:"google"`
		}
		result.APM.ServerURL = apmServerURL
		result.Google.ClientID = config.Google.ClientID
		result.Google.OAuthScope = "openid email profile"
		json.NewEncoder(w).Encode(result)
	}, "GET /api/config"))

	authMiddleware := getAuthMiddleware(secureCookies, parseIDToken)

	// Authenticate endpoint: validates credentials and returns user profile
	router.GET("/api/authenticate", wrapHandler(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		logger := logger.With(traceLogFields(r.Context())...)
		authHeader := r.Header.Get("Authorization")
		var credentials string
		if authHeader != "" {
			fields := splitAuthHeader(authHeader)
			if len(fields) != 2 || fields[0] != "Bearer" {
				http.Error(w, "invalid Authorization header", http.StatusUnauthorized)
				return
			}
			credentials = fields[1]
			cookieValue, err := secureCookies.Encode(credentials)
			if err != nil {
				logger.Error("failed to encode credentials cookie", zap.Error(err))
				http.Error(w, "failed to encode cookie", http.StatusInternalServerError)
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name:     "credentials",
				Value:    cookieValue,
				Secure:   true,
				HttpOnly: true,
				Expires:  time.Now().Add(7 * 24 * time.Hour),
			})
		} else {
			cookie, err := r.Cookie("credentials")
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			credentials, err = secureCookies.Decode(cookie.Value)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
		}
		auth, err := parseIDToken(credentials)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		result := struct {
			Profile struct {
				Name    string `json:"name"`
				UserID  string `json:"id"`
				Email   string `json:"email"`
				Picture string `json:"picture"`
			} `json:"profile"`

			// GoogleAuthorized reports whether the user has authorized additional Google scopes
			GoogleAuthorized bool `json:"google_authorized"`

			// GoogleOAuthState holds a nonce to pass to the Google OAuth API
			GoogleOAuthState string `json:"google_oauth_state,omitempty"`

			// GoogleAuthorizationError holds an error message related to authorization
			GoogleAuthorizationError string `json:"google_authorization_error,omitempty"`
		}{}
		result.Profile.Name = auth.name
		result.Profile.Picture = auth.picture
		result.Profile.UserID = auth.userID
		result.Profile.Email = auth.email

		// For this simple app, we consider the user authorized after initial sign-in
		// Additional Google Drive scopes could be requested if needed
		result.GoogleAuthorized = true

		json.NewEncoder(w).Encode(result)
	}, "GET /api/authenticate"))

	// Google OAuth callback
	router.GET("/api/oauth/google", wrapHandler(authMiddleware(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		auth := authFromContext(r.Context())
		code := r.URL.Query().Get("code")
		if _, err := validateOAuthState(secureCookies, r, googleStateCookieKey); err != nil {
			http.Error(w, "invalid authorization state", http.StatusUnauthorized)
			return
		}
		token, err := oauth2ConfigForURL(googleConfig, r).Exchange(r.Context(), code)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		if err := tokens.setGoogle(r.Context(), auth.userID, token); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}), "GET /api/oauth/google"))

	// User profile endpoint (authenticated)
	router.GET("/api/user", wrapHandler(authMiddleware(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		auth := authFromContext(r.Context())
		result := struct {
			Name    string `json:"name"`
			Email   string `json:"email"`
			Picture string `json:"picture"`
			UserID  string `json:"user_id"`
		}{
			Name:    auth.name,
			Email:   auth.email,
			Picture: auth.picture,
			UserID:  auth.userID,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}), "GET /api/user"))

	// Hello endpoint (authenticated) - returns a greeting message
	router.GET("/api/hello", wrapHandler(authMiddleware(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		auth := authFromContext(r.Context())
		result := struct {
			Message   string `json:"message"`
			Timestamp string `json:"timestamp"`
			User      string `json:"user"`
		}{
			Message:   "Hello, " + auth.name + "! Welcome to the App Scaffold.",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			User:      auth.email,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}), "GET /api/hello"))

	// Data endpoint (authenticated) - returns sample table data
	router.GET("/api/data", wrapHandler(authMiddleware(func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sampleData)
	}), "GET /api/data"))

	// Admin endpoint for health checks
	router.GET("/api/admin/health", wrapHandler(basicAuthMiddleware(config.AdminSecret, func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		result := struct {
			Status    string `json:"status"`
			Timestamp string `json:"timestamp"`
		}{
			Status:    "ok",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}), "GET /api/admin/health"))

	logger.Info("starting server on :4000")
	if err := http.ListenAndServe(":4000", router); err != nil {
		logger.Fatal("server error", zap.Error(err))
	}
}

func splitAuthHeader(header string) []string {
	idx := -1
	for i, c := range header {
		if c == ' ' {
			idx = i
			break
		}
	}
	if idx == -1 {
		return []string{header}
	}
	return []string{header[:idx], header[idx+1:]}
}

func basicAuthMiddleware(secret string, h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		_, password, ok := r.BasicAuth()
		if ok && subtle.ConstantTimeCompare([]byte(password), []byte(secret)) == 1 {
			h(w, r, p)
			return
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}

func wrapHandler(handler httprouter.Handle, operation string) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		adapted := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler(w, r, p)
		})
		otelhttp.NewHandler(adapted, operation).ServeHTTP(w, r)
	}
}
