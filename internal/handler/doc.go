// Package handler provides HTTP handlers for the file-vault service,
// including user authentication, OAuth, session management, and file operations.
//
// Handlers
//
// This package contains the following main handlers:
//
//   - users.go: User authentication (SignUp, Login, Refresh, Logout)
//     * SignUp: Register new user with email/password
//     * Login: Authenticate user and issue tokens
//     * Refresh: Refresh access token using refresh token cookie
//     * Logout: Clear tokens and invalidate session
//
//   - oAuth.go: OAuth 2.0 integration (Google, GitHub)
//     * GoogleLogin: Initiate Google OAuth flow
//     * GoogleCallback: Handle Google OAuth callback
//     * GithubLogin: Initiate GitHub OAuth flow
//     * GithubCallback: Handle GitHub OAuth callback
//
//   - upload_files.go: File upload and management
//     * UploadFile: Upload any file type (max 50MB)
//     * UploadImage: Upload images only (jpg, png, gif, webp, svg, max 10MB)
//     * UploadVideo: Upload videos only (mp4, webm, mov, avi, mkv, max 500MB)
//     * GetFiles: List all user files
//     * GetFileByID: Get file details by ID
//     * DeleteFile: Delete a file
//
// Upload Endpoints
//
//     POST /upload         - Upload any file (50MB limit)
//     POST /upload/image   - Upload images only (10MB limit, jpg/png/gif/webp/svg)
//     POST /upload/video   - Upload videos only (500MB limit, mp4/webm/mov/avi/mkv)
//
// Response contains CloudFront URL for direct browser access:
//     {"id": "uuid", "message": "uploaded successfully", "url": "https://cloudfront.net/..."}
//
// File Storage
//
// Files are uploaded to AWS S3 with CloudFront distribution for fast delivery.
// Content-Type is automatically detected for proper browser handling (streaming vs download).
//
// Authentication & Security
//
//   - Access tokens: Short-lived JWT (15 min)
//   - Refresh tokens: Long-lived JWT (30 days), stored in DB as HTTP-only cookie
//   - OAuth tokens: Managed via OAuth providers
//   - Middleware injects user ID into request context
//
// File Access
//
//     - URLs use CloudFront domain (no signing required)
//     - Videos stream directly in browser (Content-Type: video/mp4)
//     - Images display directly in browser
//     - Other files download via browser
package handler