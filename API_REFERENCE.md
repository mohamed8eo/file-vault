# File Vault API - Complete Reference

## Base URL
```
http://localhost:3000
```

## Authentication
All endpoints marked with `[Requires Auth]` need JWT token in Authorization header:
```
Authorization: Bearer <access_token>
```

## Response Format
All responses are JSON. Success responses have `2xx` status codes, errors have `4xx` or `5xx`.

---

## Authentication Endpoints

### 1. Sign Up
**POST** `/auth/sign-up`

Register a new user account.

**Request:**
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "SecurePassword123!"
}
```

**Response (201 Created):**
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "access_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Cookies Set:**
- `refresh_token`: HTTP-only cookie (30 days)

**Status Codes:**
- `201`: User created successfully
- `400`: Missing/invalid fields
- `409`: Email already exists
- `500`: Server error

---

### 2. Login
**POST** `/auth/login`

Authenticate user with email and password.

**Request:**
```json
{
  "email": "john@example.com",
  "password": "SecurePassword123!"
}
```

**Response (200 OK):**
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "access_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Cookies Set:**
- `refresh_token`: HTTP-only cookie (30 days, replaces old token)

**Status Codes:**
- `200`: Login successful
- `400`: Missing fields
- `401`: Invalid credentials
- `500`: Server error

---

### 3. Refresh Token
**POST** `/auth/refresh`

Get a new access token using refresh token cookie.

**Request:**
- No body required
- Sends `refresh_token` cookie automatically

**Response (200 OK):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Cookies Set:**
- `refresh_token`: New HTTP-only cookie (30 days)

**Status Codes:**
- `200`: Token refreshed
- `401`: Invalid/expired refresh token
- `500`: Server error

---

### 4. Logout
**POST** `/auth/logout`

Invalidate refresh token and logout user.

**[Requires Auth]**

**Request:**
- No body required
- Sends `refresh_token` cookie automatically

**Response (200 OK):**
```json
{
  "message": "logged out successfully"
}
```

**Cookies Cleared:**
- `refresh_token`: Set to empty with Max-Age=0

**Status Codes:**
- `200`: Logout successful
- `401`: Missing refresh token
- `500`: Server error

---

### 5. Google OAuth - Initiate
**GET** `/auth/google?cli=<true|false>`

Redirect to Google OAuth consent screen.

**Query Parameters:**
- `cli` (optional): Set to `true` for CLI client (affects redirect)

**Response:**
- Redirects to Google OAuth (HTTP 302)

---

### 6. Google OAuth - Callback
**GET** `/auth/google/callback?code=<auth_code>&state=<state>`

Handle Google OAuth callback.

**Query Parameters:**
- `code` (required): Authorization code from Google
- `state` (optional): State from initiate request

**Response (200 OK):**
If web client:
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "access_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

If CLI client: Redirects to local callback handler

**Cookies Set:**
- `refresh_token`: HTTP-only cookie (30 days)

**Status Codes:**
- `200`: OAuth successful
- `400`: Missing code parameter
- `500`: OAuth exchange failed

---

### 7. GitHub OAuth - Initiate
**GET** `/auth/github?cli=<true|false>`

Redirect to GitHub OAuth authorization.

**Query Parameters:**
- `cli` (optional): Set to `true` for CLI client

**Response:**
- Redirects to GitHub OAuth (HTTP 302)

---

### 8. GitHub OAuth - Callback
**GET** `/auth/github/callback?code=<auth_code>&state=<state>`

Handle GitHub OAuth callback.

**Query Parameters:**
- `code` (required): Authorization code from GitHub
- `state` (optional): State from initiate request

**Response (200 OK):**
```json
{
  "name": "github_username",
  "email": "user@example.com",
  "access_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Cookies Set:**
- `refresh_token`: HTTP-only cookie (30 days)

**Status Codes:**
- `200`: OAuth successful
- `400`: Missing code parameter
- `500`: OAuth exchange failed

---

## File Upload Endpoints

### 9. Upload Generic File
**POST** `/upload`

**[Requires Auth]**

Upload any file (max 50MB).

**Request:**
- Content-Type: `multipart/form-data`
- Form field: `file` (file to upload)

**Example (curl):**
```bash
curl -X POST http://localhost:3000/upload \
  -H "Authorization: Bearer <token>" \
  -F "file=@document.pdf"
```

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "uploaded successfully",
  "url": "https://d1234567890.cloudfront.net/files/1234567890-document.pdf"
}
```

**Status Codes:**
- `200`: Upload successful
- `400`: Invalid file or missing multipart data
- `401`: Unauthorized
- `413`: File too large (max 50MB)
- `429`: Rate limit exceeded
- `500`: Server error

---

### 10. Upload Image
**POST** `/upload/image`

**[Requires Auth]**

Upload image file (max 10MB, JPEG/PNG/GIF/WebP/SVG).

**Request:**
- Content-Type: `multipart/form-data`
- Form field: `file` (image file)

**Supported Formats:**
- JPG/JPEG
- PNG
- GIF
- WebP
- SVG

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "uploaded successfully",
  "url": "https://d1234567890.cloudfront.net/images/1234567890-photo.jpg"
}
```

**Status Codes:**
- `200`: Upload successful
- `400`: Invalid file or missing multipart data
- `401`: Unauthorized
- `406`: File type not allowed
- `413`: File too large (max 10MB)
- `429`: Rate limit exceeded
- `500`: Server error

---

### 11. Upload Video
**POST** `/upload/video`

**[Requires Auth]**

Upload video file (max 500MB, MP4/WebM/MOV/AVI/MKV).

**Request:**
- Content-Type: `multipart/form-data`
- Form field: `file` (video file)

**Supported Formats:**
- MP4
- WebM
- MOV
- AVI
- MKV

**Response (200 OK):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "message": "uploaded successfully",
  "url": "https://d1234567890.cloudfront.net/videos/1234567890-movie.mp4"
}
```

**Status Codes:**
- `200`: Upload successful
- `400`: Invalid file or missing multipart data
- `401`: Unauthorized
- `406`: File type not allowed
- `413`: File too large (max 500MB)
- `429`: Rate limit exceeded
- `500`: Server error

---

## File Management Endpoints

### 12. List Files
**GET** `/files?limit=20&offset=0&sort=date&type=&page=1`

**[Requires Auth]**

Get list of user's files with pagination, sorting, and filtering.

**Query Parameters:**
- `limit` (optional, default=20, max=100): Number of files to return
- `offset` (optional, default=0): Number of files to skip
- `page` (optional, default=1): Page number (overrides offset if set)
- `sort` (optional, default="date"): Sort by `date`, `name`, or `size`
- `type` (optional): Filter by `image`, `video`, or `document`

**Examples:**
```
GET /files?limit=50&offset=0&sort=date
GET /files?page=2&limit=20
GET /files?type=image&sort=name
```

**Response (200 OK):**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "file_name": "document.pdf",
    "file_url": "https://d1234567890.cloudfront.net/files/1234567890-document.pdf",
    "file_size": 1048576,
    "created_at": "2026-04-30T10:30:00Z"
  },
  {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "file_name": "photo.jpg",
    "file_url": "https://d1234567890.cloudfront.net/images/1234567890-photo.jpg",
    "file_size": 2097152,
    "created_at": "2026-04-30T09:15:00Z"
  }
]
```

**Status Codes:**
- `200`: Files retrieved
- `400`: Invalid parameters
- `401`: Unauthorized
- `500`: Server error

---

### 13. Search Files
**GET** `/files/search?q=<query>&limit=20`

**[Requires Auth]**

Search files by name (case-insensitive substring match).

**Query Parameters:**
- `q` (required, min 2 chars): Search query
- `limit` (optional, default=20, max=100): Number of results

**Examples:**
```
GET /files/search?q=photo
GET /files/search?q=document&limit=50
```

**Response (200 OK):**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "file_name": "photo_summer.jpg",
    "file_url": "https://d1234567890.cloudfront.net/images/1234567890-photo-summer.jpg",
    "file_size": 2097152,
    "created_at": "2026-04-30T10:30:00Z"
  }
]
```

**Status Codes:**
- `200`: Search completed
- `400`: Query too short or invalid
- `401`: Unauthorized
- `500`: Server error

---

### 14. Get Storage Statistics
**GET** `/files/stats`

**[Requires Auth]**

Get storage usage statistics (total files, total size, breakdown by type).

**Request:**
- No query parameters

**Response (200 OK):**
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

**Note:**
- Sizes are in bytes
- Types: images, videos, documents
- CLI command provides formatted table view

**Status Codes:**
- `200`: Stats retrieved
- `401`: Unauthorized
- `500`: Server error

---

### 15. Get File Details
**GET** `/files/{id}`

**[Requires Auth]**

Get file details by ID (including CloudFront URL).

**Path Parameters:**
- `id` (required): File UUID

**Example:**
```
GET /files/550e8400-e29b-41d4-a716-446655440000
```

**Response (200 OK):**
```json
{
  "message": "Get file URL Successfully",
  "file_url": "https://d1234567890.cloudfront.net/images/1234567890-photo.jpg",
  "file_name": "photo.jpg",
  "file_size": 2097152
}
```

**Status Codes:**
- `200`: File found
- `401`: Unauthorized
- `404`: File not found or not owned by user
- `500`: Server error

---

### 16. Delete File
**DELETE** `/files/{id}`

**[Requires Auth]**

Delete a single file from S3 and database.

**Path Parameters:**
- `id` (required): File UUID

**Example:**
```
DELETE /files/550e8400-e29b-41d4-a716-446655440000
```

**Response (204 No Content):**
- No response body

**Status Codes:**
- `204`: File deleted successfully
- `401`: Unauthorized
- `404`: File not found or not owned by user
- `500`: Server error

---

### 17. Bulk Delete Files
**POST** `/files/delete`

**[Requires Auth]**

Delete multiple files at once.

**Request:**
```json
{
  "ids": [
    "550e8400-e29b-41d4-a716-446655440000",
    "550e8400-e29b-41d4-a716-446655440001"
  ]
}
```

**Response (200 OK):**
```json
{
  "message": "deleted successfully",
  "deleted_count": 2
}
```

**Status Codes:**
- `200`: Files deleted
- `400`: Invalid request body
- `401`: Unauthorized
- `500`: Server error

---

## Share Links Endpoints

### 19. Create Share Link
**POST** `/files/{id}/share`

**[Requires Auth]**

Create a shareable link for a file with optional expiration, password, and download limits.

**Path Parameters:**
- `id` (required): File UUID

**Request:**
```json
{
  "expires_at": "2025-05-15T12:00:00Z",
  "password": "optionalpassword123",
  "max_downloads": 10
}
```

**Note:** All fields are optional. If `expires_at` is not provided, link never expires. If `password` is not provided, no password protection. If `max_downloads` is not provided, unlimited downloads.

**Response (201 Created):**
```json
{
  "message": "share link created",
  "share_url": "http://localhost:3000/s/fv-abc123def456ghi78",
  "token": "fv-abc123def456ghi78",
  "expires_at": "2025-05-15T12:00:00Z",
  "max_downloads": 10,
  "file_name": "document.pdf"
}
```

**Status Codes:**
- `201`: Share link created
- `400`: Invalid request
- `401`: Unauthorized
- `404`: File not found
- `500`: Server error

---

### 20. List Share Links
**GET** `/files/{id}/shares`

**[Requires Auth]**

Get all share links for a specific file.

**Path Parameters:**
- `id` (required): File UUID

**Response (200 OK):**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "token": "fv-abc123def456ghi78",
    "share_url": "http://localhost:3000/s/fv-abc123def456ghi78",
    "expires_at": "2025-05-15T12:00:00Z",
    "has_password": true,
    "max_downloads": 10,
    "download_count": 3,
    "created_at": "2026-05-01T10:30:00Z"
  }
]
```

**Status Codes:**
- `200`: Success
- `401`: Unauthorized
- `404`: File not found

---

### 21. Access Shared File
**GET** `/share/{token}`

**[No Auth Required - Public]**

Access a shared file via token.

**Path Parameters:**
- `token` (required): Share link token

**Query Parameters (optional):**
- `password` (if link is password protected): Send as query param or body

**Response (200 OK):**
```json
{
  "file_name": "document.pdf",
  "file_size": 1048576,
  "file_url": "https://d1234567890.cloudfront.net/files/document.pdf",
  "downloads": 4,
  "expires_at": "2025-05-15T12:00:00Z"
}
```

**Status Codes:**
- `200`: Success
- `400`: Invalid token
- `401`: Password required
- `404`: Share link not found
- `410`: Share link expired

---

### 22. Delete Share Link
**DELETE** `/share/{id}`

**[Requires Auth]**

Revoke a share link.

**Path Parameters:**
- `id` (required): Share link UUID

**Response (200 OK):**
```json
{
  "message": "share link deleted"
}
```

**Status Codes:**
- `200`: Success
- `401`: Unauthorized
- `404`: Share link not found

---

## Health Check

### 18. Health Check
**GET** `/health`

Health check endpoint (no authentication).

**Response (200 OK):**
```
OK
```

**Status Codes:**
- `200`: Service is healthy
- `500`: Service unhealthy

---

## Rate Limiting

Rate limits are applied per endpoint:

| Endpoint Type | Limit | Window |
|---------------|-------|--------|
| Login/Auth | 10 requests | 1 minute |
| Upload | 20 requests | 1 minute |
| General/Files | 100 requests | 1 minute |

**Headers on Rate Limit:**
- `429 Too Many Requests`
- `Retry-After: 60` (recommended wait time in seconds)

---

## Token Specifications

### Access Token
- **Expiration**: 15 minutes
- **Format**: JWT (JSON Web Token)
- **Algorithm**: HS256
- **Payload**: `{sub: user_id, type: "access"}`
- **Storage**: In memory (client-side)

### Refresh Token
- **Expiration**: 30 days
- **Format**: JWT
- **Algorithm**: HS256
- **Payload**: `{sub: user_id, type: "refresh"}`
- **Storage**: HTTP-only cookie (cannot be accessed by JavaScript)
- **Validation**: Checked against database on refresh

---

## Error Response Format

**Standard Error Response:**
```json
{
  "error": "Descriptive error message"
}
```

**Common Error Messages:**
- `unauthorized` - Missing or invalid token
- `file too large` - Exceeds size limit
- `file type not allowed` - Invalid file format
- `email already exists` - Duplicate email
- `invalid email or password` - Wrong credentials
- `too many requests` - Rate limited

---

## Curl Examples

### Sign Up
```bash
curl -X POST http://localhost:3000/auth/sign-up \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com",
    "password": "SecurePassword123!"
  }'
```

### Upload File
```bash
curl -X POST http://localhost:3000/upload \
  -H "Authorization: Bearer <token>" \
  -F "file=@document.pdf"
```

### List Files
```bash
curl -X GET "http://localhost:3000/files?limit=20&sort=date" \
  -H "Authorization: Bearer <token>"
```

### Search Files
```bash
curl -X GET "http://localhost:3000/files/search?q=photo" \
  -H "Authorization: Bearer <token>"
```

### Get Stats
```bash
curl -X GET http://localhost:3000/files/stats \
  -H "Authorization: Bearer <token>"
```

### Delete File
```bash
curl -X DELETE http://localhost:3000/files/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer <token>"
```

---

## Swagger UI

Interactive API documentation available at:
```
http://localhost:3000/docs
```

