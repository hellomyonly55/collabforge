# CollabForge

[简体中文](README.md) | **English**

**Team Collaboration & Project Management Platform**

CollabForge brings team management, project boards, task assignments, comments, activity logs, and progress analytics into one workspace.

The backend uses Go, Gin, GORM, MySQL, and Redis. The frontend uses Vue 3, TypeScript, Element Plus, and Pinia. The application can be deployed with Docker Compose.

## Features

| Area | Capabilities |
|---|---|
| Authentication | Registration, login/logout, profiles, password changes, bcrypt hashing, Redis sessions, and rate limiting |
| Teams | Create and edit teams, invite codes, member roles, ownership transfer, leaving, and archiving |
| Projects | Create, edit, and archive projects with permissions inherited from current team membership |
| Tasks | Four-column boards, drag-and-drop and button-based ordering, one assignee per task, priorities, dates, filters, and personal task lists |
| Collaboration | Plain-text comments, team and project activity search, and optimistic concurrency control |
| Analytics | Task distributions by status, priority, and assignee, completion rates, and overdue counts |
| Administration | Secure administrator setup, user suspension and reactivation, session revocation, team archiving, and platform audit logs |
| Testing | Integration tests, browser end-to-end tests, multi-user workflows, and dependency failure recovery checks |

## Quick start

Open the project root containing `compose.yaml`. Start Docker Desktop or your Docker engine. Docker Compose v2 is required; Go is not required on the host for a full Docker deployment.

If Node.js is available, generate local configuration without overwriting an existing `.env`:

```sh
node scripts/init-env.mjs
docker compose up --build -d
docker compose ps -a
curl -i http://localhost:8080/api/v1/health/ready
```

Without Node.js, copy `.env.example` to `.env` and set `MYSQL_PASSWORD` and `MYSQL_ROOT_PASSWORD` to two different, nonempty passwords before running the Compose commands. Keep the original database passwords when reusing existing volumes: changing `.env` does not change credentials stored in MySQL.

Open [CollabForge](http://localhost:8080) or the [system health page](http://localhost:8080/health). The `migrate` service exiting with code 0 is expected. The readiness endpoint should return HTTP 200 with `mysql`, `redis`, and `migrations` all reporting `ok`.

The first build downloads images and dependencies. If ports are occupied, change `WEB_PORT`, `MYSQL_PORT`, or `REDIS_PORT` in `.env`. When running Go locally, also update `REDIS_ADDR` as needed. Use `localhost` in the browser to match the configured origin.

Stop the application while keeping its data:

```sh
docker compose down
# Restart using the existing MySQL and Redis volumes:
docker compose up -d
```

The current database migration version is **12**. Upgrade an existing deployment with `docker compose up --build -d`.

## Accounts and administration

Users can register, sign in, and create a team or join one with an invite code.

Initialize an administrator in a new environment:

```sh
docker compose exec backend /app/admin --email your-admin@example.com --nickname Administrator
```

Use an email address that has not already been registered. Enter the password twice at the hidden terminal prompts. There is no default administrator password, and the command does not overwrite an existing account. Automation can use the explicit `--password-stdin` option; passwords are not accepted as command-line arguments.

Keep administrator credentials locally and out of Git. New environments do not create an administrator automatically. After an administrator signs in, the administration area appears in the sidebar.

## Local development

Requirements: Go 1.26+, Node.js 22.12+, npm, and a running Docker engine. The frontend Docker image uses Node.js 24.

From the project root:

```sh
# Load only a trusted .env file that you maintain.
set -a
. ./.env
set +a
docker compose up -d mysql redis
cd backend
go mod download
go run ./cmd/migrate
go run ./cmd/server
```

In another terminal, from the project root:

```sh
cd frontend
npm ci
npm run dev
```

Open http://localhost:5173. Vite proxies `/api` requests to `127.0.0.1:8081` under the same browser origin. In Compose, the backend application port is not exposed to the host. The local Go server does not load `.env` automatically.

## Testing and verification

Install the frontend dependencies and test browser, then run the verification script:

```sh
cd frontend
npm ci
npx playwright install chromium
cd ..
./scripts/verify.sh
```

You can use an installed Chrome browser instead of downloading Chromium. For example, on macOS:

```sh
PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH='/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' ./scripts/verify.sh
```

The script runs local configuration initialization tests, Compose validation, Go unit tests, static checks and builds, OpenAPI and documentation validation, frontend type checks and builds, integration tests, browser regression tests, multi-user workflows, and dependency recovery checks. It requires Go, Node.js, and Docker on the host and exits with a nonzero status on failure.

You can also run individual checks:

```sh
(cd backend && go test ./... && go vet ./... && go run ./cmd/checkdocs)
(cd frontend && npm run build)
./scripts/test-integration.sh
./scripts/test-e2e.sh
./scripts/test-workflow.sh
```

Go integration tests use an isolated `taskflow_test` database and Redis environment on ports 13306 and 16379. Browser tests use a separate application on port 18080 with their own MySQL and Redis services, which are stopped afterward. The three-account workflow starts with fresh test state, keeping registration rate limiting active without affecting the main database. Set `E2E_WEB_PORT` to change the browser test port.

Running `npm run test:e2e` directly writes data to the selected target. Use the scripts for full verification, including CLI administrator initialization for administration scenarios.

Without `TASKFLOW_INTEGRATION`, ordinary `go test` runs explicitly skip tests that require external services. On failure, inspect the exit status, the failing test, and error context in `frontend/test-results`. See the [test matrix](docs/test-matrix.md) for coverage details.

## Business rules and permissions

- Team roles (`owner`, `admin`, `member`) are separate from platform roles (`admin`, `user`). Platform administrators do not automatically receive access to team tasks.
- Resource queries validate current membership and team ownership. Leaving or being removed from a team clears the user's assignments on its nondeleted tasks. Owners must transfer ownership before leaving.
- Archived teams and projects retain their history and become read-only for normal operations. Soft-deleted tasks are excluded from lists and statistics. Comment authors can edit their comments; authors and team administrators can delete them.
- Tasks and comments have separate version fields. Reordering increments the versions of affected tasks. Board pagination uses a project revision to prevent combining different snapshots.
- Business changes and their activity records share a transaction. Suspending or reactivating a user increments the database session version, so old sessions remain invalid even if Redis cleanup fails.
- IDs are returned as strings. Audit timestamps use UTC, and due dates use SQL `DATE`. Overdue status follows the current date in `Asia/Shanghai`; tasks due today or already completed are not overdue.
- Sessions last seven days and use HttpOnly/SameSite cookies. Password changes revoke sessions on all devices. Credentials are not stored in localStorage. Updates require a refresh or a retry after a conflict because there are no real-time push updates.

## Documentation

The detailed documents below are currently in Chinese; the OpenAPI specification describes the API contract.

- [Architecture and core Go files](docs/architecture.md)
- [Database](docs/database.md)
- [API specification](docs/openapi.yaml)
- [Features and test coverage](docs/test-matrix.md)

## Troubleshooting and scope

Start Docker if the daemon is unavailable. For missing configuration, check `.env` and whether it has been loaded into your shell. For HTTP 502, check the backend and Nginx; for readiness HTTP 503, check dependencies and migrations. For origin HTTP 403, use the configured origin consistently rather than mixing `localhost` and `127.0.0.1`. HTTP 429 indicates rate limiting; wait before retrying. Integration and browser tests use isolated environments. If an old page fails to load assets after an update, reload the page.

The current version does not support real-time push updates, email, attachments, threaded replies, Markdown, custom workflows, historical trends, exports, restoring archived or deleted resources, online role elevation, or administrator resets of other users' passwords. Platform audit records are stored in the database, but there is no audit search page yet. Assignee statistics return all groups in one response, and whole-column reordering is intended for small to medium projects.
