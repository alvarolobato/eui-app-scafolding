# Agent loop protocol

You are executing a spec in an iterative, multi-session loop.

## Required behavior
- Pick the **first unchecked** task(s) in `## Tasks`.
- Implement **up to 50 tasks** per turn (including their acceptance checks).
- **Before marking tasks complete**: If the tasks involve code changes, run all tests, linting, type-checking, and formatting commands for the repo. Fix any failures before proceeding. If the spec includes a dedicated "run checks" task, defer to that task instead.
- Update the spec:
  - Mark all completed tasks as done (`[x]`).
  - Append discoveries/gotchas to `## Additional Context`.
  - Adjust remaining tasks if reality differs (split/merge/reword as needed).
  - Update `## Status`:
    - `in-progress` when the first implementation task begins
    - `done` only when the spec's "Definition of done" is met (and all tasks needed to satisfy it are complete)
- Exit after updating the spec so the next fresh session can continue.

## Spec source: File vs GitHub Issue

**File-based specs** (prompt does NOT contain `<!-- Issue #`):
- Update the local spec file directly
- When `done`, update the status in `spec/README.md`

**GitHub Issue specs** (prompt contains `<!-- Issue #N -->` and `<!-- URL: ... -->`):
- The spec lives in the GitHub issue body, NOT a local file
- Update the issue body using `gh issue edit`:
  ```bash
  gh issue edit <number> --repo <owner/repo> --body-file <temp-file>
  ```
- To update: read current body, modify it, write to temp file, then edit
- Extract owner/repo from the `<!-- URL: https://github.com/<owner>/<repo>/issues/<N> -->` comment
- Do NOT update local spec files or `spec/README.md` for issue-based specs
- The next iteration will fetch the updated issue body from GitHub

## Asking questions (GitHub Issue specs only)

Sometimes you need clarification before proceeding. Use this protocol to pause and ask the user.

**When to ask:**
- Ambiguity in requirements that could lead to wrong implementation
- Missing information needed to proceed
- Need user decision between multiple valid approaches
- Discovered a blocker that requires user input

**How to ask:**
1. Post a comment on the issue with the `## 🤖 Ralph says...` format:
   ```bash
   gh issue comment <number> --repo <owner/repo> --body "$(cat <<'EOF'
   ## 🤖 Ralph says...

   <Your question or clarification request>

   **Options:** (if applicable)
   1. Option A - description
   2. Option B - description

   ---
   Blocked on: Task N (task name)
   EOF
   )"
   ```

2. Update the issue body to add a "Blocked on" section after `## Status`:
   ```markdown
   ## Status
   agent-question

   ## Blocked on
   - **Task**: N (task name)
   - **Question**: Brief summary of what you're asking
   - **Waiting for**: User reply on this issue
   ```

3. Update labels:
   ```bash
   gh issue edit <number> --repo <owner/repo> --remove-label "status:in-progress" --add-label "status:agent-question"
   ```

4. Exit the session. The user will reply to your comment and set `status:ready` when you should continue.

**Example question comment:**
```
## 🤖 Ralph says...

I need clarification on the authentication approach. Should we use JWT tokens or session cookies for the login endpoint?

**Options:**
1. JWT - stateless, better for APIs
2. Session cookies - simpler, better for web apps

---
Blocked on: Task 3 (Implement login endpoint)
```

**Resumption:** When the user replies and sets `status:ready`, the next session will see their answer in the issue comments. Read recent comments to find the user's response before continuing.

## Adding tasks (GitHub Issue specs only)

Sometimes you discover necessary work that wasn't in the original spec. Add tasks as needed and continue execution.

**When to add tasks:**
- Discovered prerequisite work not anticipated in the original spec
- Found a bug or issue that must be fixed before continuing
- Realized a task needs to be split into multiple steps
- Identified missing tests, documentation, or cleanup work

**How to add tasks:**
1. Insert the new task(s) in the appropriate position in the `## Tasks` list. Use your judgment on placement:
   - If it's a prerequisite, insert before the task that depends on it
   - If it's follow-up work, insert after the related task
   - If it's cleanup/polish, add near the end (before final commit/PR tasks)

2. Add the `agent-added-tasks` label for audit trail:
   ```bash
   gh issue edit <number> --repo <owner/repo> --add-label "agent-added-tasks"
   ```

3. Post a comment explaining what was added and why:
   ```bash
   gh issue comment <number> --repo <owner/repo> --body "$(cat <<'EOF'
   ## 🤖 Ralph says...

   I've added new task(s) to the spec:

   **Added tasks:**
   - Task N: <description> - <reason why it's needed>

   **Placement rationale:** <why you put it where you did>
   EOF
   )"
   ```

4. Continue with execution. The user will review task additions during PR review.

**Example task-addition comment:**
```
## 🤖 Ralph says...

I've added new task(s) to the spec:

**Added tasks:**
- Task 4: Add database migration for new user_preferences table - The existing schema doesn't support the preferences feature; we need a migration before implementing the API endpoint

**Placement rationale:** Inserted before Task 5 (Implement preferences endpoint) since the endpoint depends on this table existing.
```

**Note:** The `agent-added-tasks` label remains as an audit trail showing this spec had tasks added during execution. The user reviews all changes during PR review.


---

# Application Scaffolding from ZaaS

## Status
ready

## Context
- **Problem**: Need a generic, production-ready application scaffolding based on the zaas project (https://github.com/alvarolobato/zaas) that can be reused to create different applications. The zaas project is a Zoom-archival service with excellent infrastructure setup, but we need to strip out the domain-specific logic and create a generic template that retains all build, deployment, and observability infrastructure.
- **Scope**: 
  - **In scope**: Copy and adapt zaas project structure, retain build/deployment/Tilt setup, keep backend (Go) and frontend (React), maintain ancillary services (Elasticsearch, Kibana, APM), implement basic Google authentication, create simple "Hello World" functionality with backend API integration and data table display
  - **Out of scope**: Zoom/Slack/Google Drive archival services, meeting management features, drive folder selection, production deployment configuration (we'll focus on local development setup)
- **Constraints**: 
  - Must maintain compatibility with existing build tooling (ko, Vite, Docker)
  - Must preserve Tilt development workflow
  - Must keep OpenTelemetry instrumentation
  - Should use Go for backend, React for frontend (matching zaas stack)
  - Frontend should use Elastic UI (EUI) components
- **Repo touchpoints**: All files in the repository - this is a greenfield scaffolding creation
- **Formats impacted**: None (new project)
- **Definition of done**: 
  - Application builds successfully (backend, frontend)
  - Tilt brings up local environment with all services
  - User can authenticate with Google
  - Frontend displays "Hello World" page with data from backend API
  - Table displays sample data fetched from backend
  - All observability stack (ES, Kibana, APM) running and configured
  - README documents setup and development workflow
- **Additional user input**: 
  - Remove archival services (very specific to zaas)
  - Keep backend, frontend, and ancillary services (Kibana, ES, etc.)
  - Application should accept Google authentication
  - Show hello screen (will be replaced with actual application later)
  - Hello world data should come from backend API call
  - Include a table with data from backend

## Tasks

### Phase 1: Project Structure & Base Setup
- [ ] 1) Create root project structure and copy base configuration files (owner: agent)
  - **Change**: Create directory structure and copy non-domain-specific files from zaas
  - **Files**: 
    - `/` (root): `.gitignore`, `Makefile`, `Tiltfile`
    - `/deploy/`: Helm charts, dev configs, Tilt helpers
    - Create placeholder directories: `/backend`, `/frontend`, `/docs`
  - **Acceptance**: Directory structure exists, Tiltfile present, deploy/ folder structure matches zaas pattern
  - **Spec update**: Mark done, note any directory structure decisions

- [ ] 2) Set up Elasticsearch, Kibana, APM deployment configurations (owner: agent)
  - **Change**: Copy ECK deployment YAML files and initialization scripts from zaas (elasticsearch.yaml, kibana.yaml, apm-server.yaml, init_es.sh)
  - **Files**: 
    - `/deploy/dev/elasticsearch.yaml`
    - `/deploy/dev/kibana.yaml`
    - `/deploy/dev/apm-server.yaml`
    - `/deploy/dev/init_es.sh` (modify to create generic indices instead of zaas-specific)
  - **Acceptance**: Files present, init_es.sh creates generic "app-data" index instead of zaas-specific indices
  - **Spec update**: Mark done, document index names chosen

- [ ] 3) Create Helm chart structure (owner: agent)
  - **Change**: Copy zaas Helm chart, remove archiver cronjob, simplify to backend/frontend/proxy only
  - **Files**: 
    - `/deploy/helm/Chart.yaml`
    - `/deploy/helm/values.yaml`
    - `/deploy/helm/templates/backend-deployment.yaml`
    - `/deploy/helm/templates/frontend-deployment.yaml`
    - `/deploy/helm/templates/secrets.yaml`
    - `/deploy/helm/templates/service.yaml`
    - Remove: archiver-related templates
  - **Acceptance**: Helm chart structure exists, `helm lint` passes, no archiver references
  - **Spec update**: Mark done, note any values.yaml simplifications

### Phase 2: Backend Service
- [ ] 4) Create backend Go module and base structure (owner: agent)
  - **Change**: Create Go backend with HTTP server, config loading, logging, and OpenTelemetry setup (copy otel.go, basic patterns from zaas backend)
  - **Files**:
    - `/backend/go.mod`, `/backend/go.sum`
    - `/backend/main.go` (simplified from zaas - HTTP router, middleware, config)
    - `/backend/otel.go` (copy from zaas)
    - `/backend/config.go` (simplified - Google auth only, ES config, admin secret)
    - `/backend/Dockerfile` (copy from zaas backend)
  - **Acceptance**: `go mod tidy` succeeds, builds successfully, no compilation errors
  - **Spec update**: Mark done, document dependencies chosen

- [ ] 5) Implement Google OAuth authentication in backend (owner: agent)
  - **Change**: Implement Google OAuth flow (copy oauth patterns from zaas, remove Zoom/Slack)
  - **Files**:
    - `/backend/auth.go` (OAuth state handling, token validation, middleware)
    - `/backend/main.go` (add OAuth endpoints: /api/oauth/google, /api/config, /api/user)
  - **Acceptance**: Backend exposes Google OAuth endpoints, can exchange auth code for token, validates JWT, stores user session in secure cookie
  - **Spec update**: Mark done, note OAuth scopes used

- [ ] 6) Create basic API endpoints (owner: agent)
  - **Change**: Implement simple API endpoints for frontend to consume
  - **Files**: `/backend/main.go` (add endpoints)
  - **Endpoints to implement**:
    - `GET /api/config` - returns frontend configuration (Google client ID)
    - `GET /api/user` - returns authenticated user profile
    - `GET /api/hello` - returns "Hello World" message (authenticated)
    - `GET /api/data` - returns sample table data array (authenticated)
  - **Acceptance**: All endpoints return proper JSON, auth middleware protects /api/hello and /api/data, curl tests work locally
  - **Spec update**: Mark done, document API response formats

- [ ] 7) Implement Elasticsearch integration for session storage (owner: agent)
  - **Change**: Create ES client, implement basic document storage for user sessions (simplified from zaas tokenstorage.go)
  - **Files**:
    - `/backend/storage.go` (ES client initialization, user session CRUD operations)
    - `/backend/main.go` (wire up ES client)
  - **Acceptance**: Backend can connect to ES, store/retrieve user sessions, graceful handling if ES unavailable
  - **Spec update**: Mark done, document ES index schema

### Phase 3: Frontend Application
- [ ] 8) Create React frontend structure with Vite (owner: agent)
  - **Change**: Initialize React app with Vite, copy zaas frontend structure and EUI dependencies
  - **Files**:
    - `/frontend/package.json` (copy from zaas, verify EUI, React Router versions)
    - `/frontend/vite.config.js` (copy from zaas)
    - `/frontend/index.html`
    - `/frontend/src/main.jsx` (copy from zaas)
    - `/frontend/src/icons_hack.jsx` (copy from zaas - needed for EUI icons)
    - `/frontend/Dockerfile` (copy from zaas)
    - `/frontend/nginx.conf` (copy from zaas if present)
  - **Acceptance**: `yarn install` succeeds, `npm run dev` starts development server, EUI styles load correctly
  - **Spec update**: Mark done, document Node/npm versions required

- [ ] 9) Implement authentication context and Google OAuth flow (owner: agent)
  - **Change**: Create React context for authentication, implement Google OAuth flow
  - **Files**:
    - `/frontend/src/auth.jsx` (simplified from zaas - only Google OAuth, profile management)
    - `/frontend/src/config.js` (configuration fetching utility)
  - **Acceptance**: Auth context provides user profile, Google authentication state, and authorization function; OAuth redirect flow works
  - **Spec update**: Mark done

- [ ] 10) Create main application layout and routing (owner: agent)
  - **Change**: Set up React Router, create page layout with EUI components
  - **Files**:
    - `/frontend/src/app.jsx` (simplified from zaas)
      - Routes: `/` (home/hello page), `/auth` (authorization page)
      - PageLayout component with EUI header, navigation
    - Remove: meeting management pages, about page (for now)
  - **Acceptance**: Application renders, navigation works, EUI header displays, routes resolve correctly
  - **Spec update**: Mark done

- [ ] 11) Build Hello World page with backend integration (owner: agent)
  - **Change**: Create home page that fetches and displays "Hello World" from backend API
  - **Files**: `/frontend/src/app.jsx` (add MainPage/HomePage component)
  - **Component requirements**:
    - Displays user profile (name, email, picture)
    - Fetches message from `GET /api/hello`
    - Displays the message in an EUI Panel or Card
    - Shows loading state while fetching
    - Handles errors gracefully with EUI CallOut
  - **Acceptance**: Page renders, displays authenticated user info, shows backend message, handles loading/error states correctly
  - **Spec update**: Mark done, note any UX decisions

- [ ] 12) Build data table page with backend data (owner: agent)
  - **Change**: Create table component that fetches and displays data from backend
  - **Files**: `/frontend/src/app.jsx` (enhance MainPage or create DataTable component)
  - **Component requirements**:
    - Fetches array of data from `GET /api/data`
    - Displays data in EUI DataTable with columns: ID, Name, Description, Created Date
    - Implements pagination (EUI built-in)
    - Includes search/filter functionality
    - Responsive layout
  - **Acceptance**: Table renders with data, pagination works, search filters data, responsive on mobile, loading states work
  - **Spec update**: Mark done

### Phase 4: Development Environment & Integration
- [ ] 13) Configure zaas-proxy for local development (owner: agent)
  - **Change**: Set up reverse proxy for local development (copy zaas-proxy from zaas)
  - **Files**:
    - `/deploy/dev/zaas-proxy/main.go` (copy from zaas, rename to app-proxy)
    - `/deploy/dev/zaas-proxy/helm/` (copy Helm chart, update naming)
    - TLS cert generation scripts
  - **Acceptance**: Proxy routes /api to backend, everything else to frontend, works on https://localhost:8443/
  - **Spec update**: Mark done

- [ ] 14) Update Tiltfile for complete local development (owner: agent)
  - **Change**: Configure Tiltfile to orchestrate all services (copy from zaas, remove archiver)
  - **Files**: `/Tiltfile`
  - **Requirements**:
    - Build backend with ko
    - Build frontend with Docker (hot reload)
    - Build proxy
    - Deploy ECK stack (ES, Kibana, APM)
    - Run init_es.sh resource
    - Create secrets (Google credentials)
    - Deploy Helm chart
    - Configure port forwarding: 8443 (proxy), 4000 (backend), 5601 (Kibana), 9200 (ES)
  - **Acceptance**: `tilt up` brings up entire stack, hot reload works for frontend, backend/proxy rebuild on changes, all resources green in Tilt UI
  - **Spec update**: Mark done, document any Tilt customizations

- [ ] 15) Create local development secrets setup (owner: agent)
  - **Change**: Document and create example secret files for local development
  - **Files**:
    - `/deploy/dev/secrets/google.yaml.example`
    - Update Tiltfile to handle missing secrets gracefully
  - **Content**: Example format showing required Google OAuth credentials (client_id, client_secret)
  - **Acceptance**: Example files present, instructions clear, Tilt handles missing files with helpful error message
  - **Spec update**: Mark done

### Phase 5: Documentation & Finalization
- [ ] 16) Create comprehensive README (owner: agent)
  - **Change**: Write README documenting setup, development, and architecture
  - **Files**: `/README.md`
  - **Sections**:
    - Project overview (generic app scaffolding)
    - Architecture diagram (backend, frontend, ancillary services)
    - Prerequisites (Docker, kind/k3d, Tilt, Go, Node.js)
    - Setup instructions (Google OAuth app creation, secrets configuration)
    - Development workflow (`tilt up`, hot reload, accessing services)
    - Project structure explanation
    - Technology stack
    - Common development tasks
  - **Acceptance**: README is clear, setup instructions work for new developer, covers common scenarios
  - **Spec update**: Mark done

- [ ] 17) Create sample data generator for backend (owner: agent)
  - **Change**: Implement sample data generation for the table endpoint
  - **Files**: `/backend/sampledata.go`
  - **Requirements**: Function that generates 50-100 sample records with varied data for table display
  - **Acceptance**: `GET /api/data` returns realistic sample data, different on each backend restart
  - **Spec update**: Mark done

- [ ] 18) Add basic backend tests (owner: agent)
  - **Change**: Create test files for key backend functionality
  - **Files**:
    - `/backend/main_test.go` (HTTP handler tests)
    - `/backend/auth_test.go` (OAuth flow tests)
  - **Acceptance**: `go test ./...` passes, basic coverage for auth and API endpoints
  - **Spec update**: Mark done

- [ ] 19) Verify complete integration and cleanup (owner: agent)
  - **Change**: End-to-end testing and cleanup
  - **Tests**:
    - Fresh `tilt up` from clean state
    - Complete OAuth flow test
    - Verify all pages render
    - Verify API integration works
    - Check Kibana/ES accessibility
    - Verify APM data flowing
    - Test hot reload (edit frontend file, verify update)
  - **Acceptance**: All integration tests pass, no errors in Tilt, application fully functional, observability working
  - **Spec update**: Update status to "done"

## Additional Context
### Key Files to Copy from zaas
- Tiltfile and deploy/tilt/*.star helpers
- Dockerfile patterns (ko for backend, multi-stage for frontend)
- OpenTelemetry setup (otel.go in both backend and archiver)
- EUI component usage patterns
- OAuth flow patterns (simplified to Google only)
- Elasticsearch integration patterns
- Helm chart structure

### Key Changes from zaas
- Remove: Zoom, Slack, Google Drive integration
- Remove: Archiver service/cronjob
- Remove: Meeting management domain logic
- Remove: Drive folder picker
- Simplify: OAuth to Google only
- Simplify: API to basic CRUD operations
- Add: Generic "Hello World" functionality
- Add: Sample data table display

### Generic Naming Convention
- Project name: "app-scaffold" or "generic-app"
- Service names: "app-backend", "app-frontend", "app-proxy"
- Elasticsearch indices: "app-data", "app-sessions"
- Kubernetes namespace: "app-local" (for local dev)

### Dependencies to Verify
**Backend (Go):**
- github.com/julienschmidt/httprouter (HTTP routing)
- github.com/elastic/go-elasticsearch/v8 (ES client)
- go.uber.org/zap (logging)
- go.opentelemetry.io/* (observability)
- golang.org/x/oauth2 (OAuth)
- github.com/golang-jwt/jwt/v4 (JWT validation)

**Frontend (Node.js/React):**
- react, react-dom
- react-router-dom
- @elastic/eui (Elastic UI components)
- @elastic/apm-rum (Real User Monitoring)
- vite

### Development Workflow Notes
- Use `kind` or `k3d` for local Kubernetes cluster
- Tilt orchestrates: build → deploy → port forward → watch
- Frontend hot reloads via Vite
- Backend rebuilds with ko (fast Go builds)
- All services accessible via localhost:8443 (TLS proxy)
- Direct backend access via localhost:4000 (for debugging)
- Kibana UI via localhost:5601
- Elasticsearch via localhost:9200
