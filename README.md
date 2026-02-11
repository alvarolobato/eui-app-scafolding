# App Scaffold

A generic, production-ready application scaffolding with Google authentication, React frontend, Go backend, and full observability stack (Elasticsearch, Kibana, APM).

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Kubernetes Cluster                       │
│                                                                  │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────┐  │
│  │  app-proxy   │────│  app-frontend│    │    app-backend   │  │
│  │  (TLS :8443) │    │   (React)    │    │      (Go)        │  │
│  └──────┬───────┘    └──────────────┘    └────────┬─────────┘  │
│         │                                          │            │
│         │  ┌───────────────────────────────────────┘            │
│         │  │                                                    │
│  ┌──────▼──▼───┐    ┌──────────────┐    ┌──────────────────┐  │
│  │Elasticsearch│────│    Kibana    │    │   APM Server     │  │
│  │   (9200)    │    │   (5601)     │    │    (8200)        │  │
│  └─────────────┘    └──────────────┘    └──────────────────┘  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

## Technology Stack

**Backend (Go):**
- HTTP routing: `julienschmidt/httprouter`
- Elasticsearch client: `elastic/go-elasticsearch/v8`
- Logging: `uber-go/zap`
- OpenTelemetry: distributed tracing
- OAuth 2.0: Google authentication

**Frontend (React):**
- UI Framework: Elastic UI (EUI)
- Build tool: Vite
- Routing: React Router
- Real User Monitoring: Elastic APM RUM

**Infrastructure:**
- Kubernetes deployment via Helm
- ECK (Elastic Cloud on Kubernetes) for ES/Kibana/APM
- Tilt for local development workflow
- ko for Go container builds

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) or [k3d](https://k3d.io/)
- [Tilt](https://docs.tilt.dev/install.html)
- [Go 1.22+](https://golang.org/dl/)
- [Node.js 20+](https://nodejs.org/)
- [ko](https://ko.build/install/)
- [Helm](https://helm.sh/docs/intro/install/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

## Setup

### 1. Create a Kubernetes Cluster

Using kind:
```bash
kind create cluster --name app-local
```

Or using k3d:
```bash
k3d cluster create app-local
```

### 2. Configure Google OAuth

1. Go to [Google Cloud Console](https://console.cloud.google.com/apis/credentials)
2. Create a new OAuth 2.0 Client ID:
   - Application type: Web application
   - Authorized JavaScript origins: `https://localhost:8443`
   - Authorized redirect URIs: `https://localhost:8443/api/oauth/google`
3. Copy the Client ID and Client Secret

### 3. Create Secrets File

```bash
cp deploy/dev/secrets/google.yaml.example deploy/dev/secrets/google.yaml
```

Edit `deploy/dev/secrets/google.yaml` with your credentials:
```yaml
client_id: "your-google-client-id.apps.googleusercontent.com"
client_secret: "your-google-client-secret"
```

### 4. Start Development Environment

```bash
tilt up
```

This will:
- Deploy ECK operator
- Create Elasticsearch, Kibana, and APM Server
- Initialize Elasticsearch indices
- Build and deploy the backend (Go)
- Build and deploy the frontend (React with hot reload)
- Deploy the TLS proxy

### 5. Access the Application

- **Application**: https://localhost:8443
- **Backend API**: http://localhost:4000
- **Kibana**: http://localhost:5601 (login: admin/changeme)
- **Elasticsearch**: http://localhost:9200
- **Tilt Dashboard**: http://localhost:10350

## Development Workflow

### Frontend Development

The frontend supports hot reloading via Vite. Edit files in `frontend/src/` and changes will appear immediately.

```bash
# Local development (outside of Tilt)
cd frontend
yarn install
yarn dev
```

### Backend Development

Backend changes trigger automatic rebuilds via ko.

```bash
# Local development (outside of Tilt)
cd backend
go run .
```

### Running Tests

```bash
# Backend tests
make test-backend

# Helm chart linting
make helm-lint
```

## Project Structure

```
.
├── backend/                 # Go backend service
│   ├── main.go             # HTTP server and routes
│   ├── auth.go             # OAuth authentication
│   ├── config.go           # Configuration management
│   ├── otel.go             # OpenTelemetry setup
│   ├── sampledata.go       # Sample data generation
│   └── ...
├── frontend/               # React frontend
│   ├── src/
│   │   ├── app.jsx        # Main application component
│   │   ├── auth.jsx       # Authentication context
│   │   └── main.jsx       # Entry point
│   ├── package.json
│   └── vite.config.js
├── deploy/
│   ├── helm/              # Helm chart for deployment
│   │   ├── Chart.yaml
│   │   ├── values.yaml
│   │   └── templates/
│   ├── dev/               # Local development configs
│   │   ├── elasticsearch.yaml
│   │   ├── kibana.yaml
│   │   ├── apm-server.yaml
│   │   ├── init_es.sh
│   │   ├── app-proxy/     # TLS reverse proxy
│   │   └── secrets/       # Local secrets (gitignored)
│   └── tilt/              # Tilt helper scripts
├── Tiltfile               # Tilt configuration
├── Makefile               # Build commands
└── README.md
```

## API Endpoints

| Endpoint | Method | Auth | Description |
|----------|--------|------|-------------|
| `/api/config` | GET | No | Frontend configuration |
| `/api/authenticate` | GET | Bearer/Cookie | Validate credentials |
| `/api/user` | GET | Yes | Get user profile |
| `/api/hello` | GET | Yes | Hello World message |
| `/api/data` | GET | Yes | Sample table data |
| `/api/oauth/google` | GET | Cookie | OAuth callback |
| `/api/admin/health` | GET | Basic | Health check |

## Elasticsearch Indices

- `app-sessions`: User session and token storage
- `app-data`: Sample application data

## Common Tasks

### View Logs

```bash
# Via Tilt dashboard
tilt up

# Or via kubectl
kubectl logs -f deployment/app-backend
kubectl logs -f deployment/app-frontend
```

### Reset Elasticsearch Data

```bash
# Delete and recreate indices
kubectl exec -it elasticsearch-es-default-0 -- curl -X DELETE http://localhost:9200/app-*
tilt trigger init-elasticsearch
```

### Rebuild All Services

```bash
tilt down
tilt up
```

## Extending the Scaffold

This scaffold is designed to be a starting point. Common extensions:

1. **Add new API endpoints**: Edit `backend/main.go`
2. **Add new pages**: Edit `frontend/src/app.jsx`
3. **Add new Elasticsearch indices**: Edit `deploy/dev/init_es.sh`
4. **Add environment variables**: Edit `deploy/helm/templates/backend/deployment.yaml`

## Troubleshooting

### "Elasticsearch API Key not set"

This is expected on first run. The `init-elasticsearch` resource creates the API key after Elasticsearch is ready.

### Certificate errors

The local proxy uses a self-signed certificate. Accept the certificate warning in your browser or:
```bash
# On macOS
security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain <cert-file>
```

### Pod not starting

Check pod status and logs:
```bash
kubectl get pods
kubectl describe pod <pod-name>
kubectl logs <pod-name>
```

## License

[Add your license here]
