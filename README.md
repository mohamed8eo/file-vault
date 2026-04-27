# File Vault

A secure file storage REST API built with Go, PostgreSQL, AWS S3, and Redis.
Supports user sign-up/login (JWT-based), secure file uploads to S3, access control, and efficient rate limiting.

## Project Structure

```
file-vault/
├── cmd/
│   └── api/
│       └── main.go           # Main entry point, bootstraps HTTP server and all modules
├── internal/
│   ├── auth/                 # Password and JWT/token logic
│   │   ├── jwt.go
│   │   └── password.go
│   ├── db/                  # sqlc-generated code: Go DB access, types, functions
│   │   ├── db.go
│   │   ├── models.go
│   │   └── *.sql.go
│   ├── handler/              # HTTP handlers for users (auth) and file uploads
│   │   ├── users.go
│   │   └── upload_files.go
│   └── middleware/           # HTTP middleware (auth, rate limit, logging, request id)
│       ├── auth.go
│       ├── rateLimit.go
│       ├── logging.go
│       └── requestid.go
├── sql/
│   ├── queries/              # Raw SQL queries (used by sqlc)
│   └── schema/               # DB schema migration files
├── go.mod / go.sum           # Dependency management
├── sqlc.yaml                 # sqlc codegen config
└── Makefile
```

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         file-vault API                          │
│                                                                 │
│  ┌──────────┐    ┌─────────────────┐    ┌───────────────────┐   │
│  │          │    │   Middleware    │    │    Handlers       │   │
│  │  Client  │───▶│  • Request ID   │───▶│  • SignUp/Login   │   │
│  │          │    │  • Logging      │    │  • Refresh/Logout │   │
│  │  curl /  │◀───│  • Auth (JWT)   │◀───│  • UploadFile     │   │
│  │ browser  │    │  • Rate limit   │    │  • GetFiles       │   │
│  └──────────┘    └────────┬────────┘    │  • DeleteFile     │   │
│                           │             └────────┬──────────┘   │
│                           │                      │              │
│                    ┌──────▼──────┐         ┌──────▼──────┐      │
│                    │    Redis    │         │  PostgreSQL │      │
│                    │  (sliding   │         │   users     │      │
│                    │   window)   │         │   tokens    │      │
│                    └─────────────┘         │   files     │      │
│                                            └─────────────┘      │
│                                                  │              │
│                                           ┌──────▼──────┐       │
│                                           │   AWS S3    │       │
│                                           │ file storage│       │
│                                           └─────────────┘       │
└─────────────────────────────────────────────────────────────────┘
```

## Login/Authentication Flow

1. **Sign Up** `/auth/sign-up`
   - User sends name, email, password.
   - Password is Argon2id-hashed; user created in DB.
   - Access token (JWT, 15m) & refresh token (JWT, 30 days) issued.
   - Refresh token is stored in DB and set as **HTTP-only cookie**.

2. **Login** `/auth/login`
   - User sends email, password.
   - Validates credentials against DB hash.
   - New tokens issued, old refresh token cleared, new one set as cookie.

3. **Refresh** `/auth/refresh`
   - Expects cookie `refresh_token`.
   - Validates JWT (refresh secret) and DB presence.
   - Issues new access/refresh tokens, updates DB and cookie.

4. **Logout** `/auth/logout`
   - Deletes refresh token from DB, clears cookie.

## API Endpoints

| Method | Endpoint | Auth | Description |
|--------|---------|------|------------|
| GET | `/` | No | Health check |
| POST | `/auth/sign-up` | No | Register new user |
| POST | `/auth/login` | No | Login user |
| POST | `/auth/refresh` | No | Refresh access token |
| POST | `/auth/logout` | No | Logout user |
| POST | `/upload` | Yes | Upload file to S3 |
| GET | `/files` | Yes | List user's files |
| GET | `/files/{id}` | Yes | Get file presigned URL |
| DELETE | `/files/{id}` | Yes | Delete file |

## Environment Variables

```bash
# Database
DB_URL=postgres://user:pass@localhost:5432/db

# JWT Secrets
ACCESS_TOKEN_SECRET=your-access-secret
REFRESH_TOKEN_SECRET=your-refresh-secret

# Redis
REDIS_URL=redis://localhost:6379

# AWS S3
S3_BUCKET=your-bucket-name
S3_REGION=us-east-1

# App
PORT=8080
IS_PRODUCTION=false

## Prerequisites
- Go 1.22+
- PostgreSQL
- Redis
- AWS account with S3 bucket

```

## Quick Start

```bash
# Install dependencies
make deps

# Run database migrations
make migrate

# Generate DB code (after changing sql/ files)
make sqlc

# Build the application
make build

# Run the server
make run

# Run tests
make test
```

## Tech Stack

- **Language**: Go 1.25+
- **Database**: PostgreSQL (via pgx/v5)
- **ORM**: sqlc
- **Cache/Rate Limit**: Redis
- **Storage**: AWS S3
- **Auth**: JWT (golang-jwt), Argon2id
- **HTTP**: stdlib net/http
