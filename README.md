# File Vault

A secure file storage REST API built with Go, PostgreSQL, AWS S3, CloudFront, and Redis.
Supports user sign-up/login (JWT + OAuth), secure file uploads to S3 with CloudFront distribution, and efficient rate limiting.

## Project Structure

```
file-vault/
├── cmd/
│   ├── api/
│   │   └── main.go           # Main entry point, HTTP server
│   └── cli/
│       ├── main.go           # CLI entry point
│       └── cmd/              # CLI commands (auth, files)
├── internal/
│   ├── auth/                 # Password hashing and JWT logic
│   │   ├── jwt.go
│   │   └── password.go
│   ├── client/               # CLI client for API calls
│   │   ├── client.go         # Token management, HTTP client
│   │   └── files.go          # File operations (upload, list, get, delete)
│   ├── db/                   # sqlc-generated DB code
│   │   ├── db.go
│   │   ├── models.go
│   │   └── *.sql.go
│   ├── handler/              # HTTP handlers
│   │   ├── users.go          # SignUp, Login, Refresh, Logout
│   │   ├── oAuth.go          # Google & GitHub OAuth
│   │   ├── upload_files.go   # File upload & management
│   │   └── doc.go            # Package documentation
│   └── middleware/           # HTTP middleware
│       ├── auth.go           # JWT validation
│       ├── rateLimit.go      # Redis-based rate limiting
│       ├── logging.go        # Request logging
│       └── requestid.go      # Request ID tracking
├── sql/
│   ├── queries/              # SQL queries for sqlc
│   └── schema/               # Database schema
├── go.mod / go.sum
└── Makefile
```

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         file-vault API                          │
│                                                                 │
│  ┌──────────┐    ┌─────────────────┐    ┌───────────────────┐   │
│  │          │    │   Middleware    │    │    Handlers       │   │
│  │  CLI /   │───▶│  • Request ID   │───▶│  • SignUp/Login   │   │
│  │  Client  │    │  • Logging      │    │  • OAuth (Google) │   │
│  │          │◀───│  • Auth (JWT)   │◀───│  • Refresh/Logout │   │
│  │  curl /  │    │  • Rate limit   │    │  • Upload(File)   │   │
│  │ browser  │    └────────┬────────┘    │  • Upload(Image) │   │
│  └──────────┘             │             │  • Upload(Video) │   │
│                           │             │  • GetFiles      │   │
│                    ┌──────▼──────┐      │  • DeleteFile    │   │
│                    │    Redis    │      └────────┬──────────┘   │
│                    │  (sliding   │               │              │
│                    │   window)   │        ┌──────▼──────┐       │
│                    └─────────────┘        │  PostgreSQL │       │
│                                            │  (users,    │       │
│                                            │   tokens,   │       │
│                                            │   files)    │       │
│                                            └─────────────┘       │
│                                                  │               │
│                                           ┌──────▼──────┐        │
│                                           │   AWS S3    │        │
│                                           │ (files)     │        │
│                                           └──────┬──────┘        │
│                                                  │               │
│                                           ┌──────▼──────┐        │
│                                           │ CloudFront  │        │
│                                           │ (streaming) │        │
│                                           └─────────────┘        │
└─────────────────────────────────────────────────────────────────┘
```

## Authentication Flow

### Email/Password

1. **Sign Up** `/auth/sign-up`
   - User sends name, email, password
   - Password is Argon2id-hashed and stored in DB
   - Issues access token (15m) and refresh token (30 days)
   - Refresh token stored in DB and set as HTTP-only cookie

2. **Login** `/auth/login`
   - User sends email, password
   - Validates credentials, issues new tokens
   - Clears old refresh token, sets new one as cookie

3. **Refresh** `/auth/refresh`
   - Expects `refresh_token` cookie
   - Validates JWT and DB presence
   - Issues new access/refresh tokens, updates DB and cookie

4. **Logout** `/auth/logout`
   - Deletes refresh token from DB, clears cookie

### OAuth 2.0 (Google & GitHub)

1. **Initiate** `/auth/google` or `/auth/github`
   - Redirects to OAuth provider
   - User authorizes the application

2. **Callback** `/auth/google/callback` or `/auth/github/callback`
   - Receives OAuth code from provider
   - Exchanges for user info
   - Creates/updates user in DB
   - Issues tokens like regular login

## API Endpoints

### Authentication

| Method | Endpoint | Auth | Description |
|--------|---------|------|-------------|
| POST | `/auth/sign-up` | No | Register new user |
| POST | `/auth/login` | No | Login user |
| POST | `/auth/refresh` | No | Refresh access token |
| POST | `/auth/logout` | No | Logout user |
| GET | `/auth/google` | No | Initiate Google OAuth |
| GET | `/auth/google/callback` | No | Google OAuth callback |
| GET | `/auth/github` | No | Initiate GitHub OAuth |
| GET | `/auth/github/callback` | No | GitHub OAuth callback |

### File Operations

| Method | Endpoint | Auth | Description |
|--------|---------|------|-------------|
| POST | `/upload` | Yes | Upload any file (max 50MB) |
| POST | `/upload/image` | Yes | Upload image (max 10MB, jpg/png/gif/webp/svg) |
| POST | `/upload/video` | Yes | Upload video (max 500MB, mp4/webm/mov/avi/mkv) |
| GET | `/files` | Yes | List files with pagination, sort, and filter |
| GET | `/files/search` | Yes | Search files by name |
| GET | `/files/stats` | Yes | Get storage statistics (total files, size by type) |
| GET | `/files/{id}` | Yes | Get file details with CloudFront URL |
| POST | `/files/delete` | Yes | Bulk delete files |
| DELETE | `/files/{id}` | Yes | Delete single file from S3 and DB |

### Query Parameters for `/files`

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | int | 20 | Number of files to return (max 100) |
| `offset` | int | 0 | Number of files to skip |
| `page` | int | 1 | Page number (calculates offset automatically) |
| `sort` | string | "date" | Sort by: `date`, `name`, `size` |
| `type` | string | "" | Filter by type: `image`, `video`, `document` |

### Health Check

| Method | Endpoint | Auth | Description |
|--------|---------|------|-------------|
| GET | `/` | No | Health check |

## File Upload Response

```json
{
  "id": "76220f7a-8cee-4443-9aee-09ecfbda9cdc",
  "message": "uploaded successfully",
  "url": "https://d30aex9619h2hb.cloudfront.net/images/1234567890-photo.jpg"
}
```

The returned URL can be used directly in browsers:
- **Videos**: Stream directly in `<video>` tag
- **Images**: Display directly in `<img>` tag
- **Other files**: Download via browser

## Storage Stats Response

```json
{
  "total_files": 42,
  "total_size": 1572864000,
  "images": {
    "count": 15,
    "size": 524288000
  },
  "videos": {
    "count": 5,
    "size": 1048576000
  },
  "documents": {
    "count": 22,
    "size": 104857600
  }
}
```

Size values are in bytes. Use the CLI's `files stats` command for a formatted table view.

## Environment Variables

```bash
# Database
DB_URL=postgres://user:pass@localhost:5432/db

# JWT Secrets (base64 encoded)
ACCESS_TOKEN_SECRET=your-access-secret
REFRESH_TOKEN_SECRET=your-refresh-secret

# Redis
REDIS_URL=redis://localhost:6379

# AWS S3
S3_BUCKET=your-bucket-name
S3_REGION=us-east-1
CLOUDFRONT_DOMAIN=d1234567890.cloudfront.net

# OAuth (optional)
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
GOOGLE_REDIRECT_URL=http://localhost:3000/auth/google/callback

GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
GITHUB_REDIRECT_URL=http://localhost:3000/auth/github/callback

# App
PORT=3000
IS_PRODUCTION=false
```

## CLI Commands

```bash
# Authentication
file-vault auth login <email> <password>
file-vault auth register <name> <email> <password>
file-vault auth logout

# File operations
file-vault files upload <path>              # Upload file
file-vault files list                       # List files
file-vault files list --sort date|name|size # Sort files
file-vault files list --type image|video|document  # Filter by type
file-vault files list --page <num>          # Paginate (page 1, 2, ...)
file-vault files list --offset <num>        # Skip offset files
file-vault files search <query>             # Search files by name
file-vault files stats                      # Show storage statistics
file-vault files get <id>                  # Get file URL (opens in browser)
file-vault files download <id> [output]     # Download file to disk
file-vault files delete <id>               # Delete single file
file-vault files delete-many <id1> <id2>... # Bulk delete files
```

## Prerequisites

- Go 1.22+
- PostgreSQL
- Redis
- AWS account with S3 bucket + CloudFront distribution

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

# Build CLI
go build -o file-vault ./cmd/cli/

# Run tests
make test
```

## Tech Stack

- **Language**: Go 1.22+
- **Database**: PostgreSQL (via pgx/v5)
- **ORM**: sqlc
- **Cache/Rate Limit**: Redis (go-redis)
- **Storage**: AWS S3
- **CDN**: CloudFront
- **Auth**: JWT (golang-jwt), Argon2id
- **OAuth**: Google, GitHub
- **HTTP**: stdlib net/http