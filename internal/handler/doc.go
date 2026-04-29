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
//     * GetFiles: List files with pagination, sort (date/name/size), filter (image/video/document)
//     * SearchFiles: Search files by name
//     * GetStorageStats: Get storage statistics (total files, size breakdown by type)
//     * GetFileByID: Get file details by ID
//     * DeleteFile: Delete a single file
//     * DeleteFiles: Bulk delete multiple files
//     * DownloadFile: Get file content for download
//
// Upload Endpoints
//
//	POST /upload         - Upload any file (50MB limit)
//	POST /upload/image   - Upload images only (10MB limit, jpg/png/gif/webp/svg)
//	POST /upload/video   - Upload videos only (500MB limit, mp4/webm/mov/avi/mkv)
//
// Response contains CloudFront URL for direct browser access:
//	{"id": "uuid", "message": "uploaded successfully", "url": "https://cloudfront.net/...", "file_size": 123456}
//
// File List Endpoints
//
//	GET /files?limit=20&offset=0&page=1&sort=date&type=image
//	GET /files/search?q=filename&limit=10
//	GET /files/stats
//
// Query Parameters:
//   - limit: Number of files to return (default 20, max 100)
//   - offset: Number of files to skip (default 0)
//   - page: Page number (calculates offset automatically)
//   - sort: Sort by "date", "name", or "size" (default "date")
//   - type: Filter by "image", "video", or "document"
//
// Storage Stats Response:
//	{
//	  "total_files": 42,
//	  "total_size": 1572864000,
//	  "images": {"count": 15, "size": 524288000},
//	  "videos": {"count": 5, "size": 1048576000},
//	  "documents": {"count": 22, "size": 104857600}
//	}
//
// Delete Endpoints
//
//	DELETE /files/{id}         - Delete single file
//	POST /files/delete         - Bulk delete (body: {"ids": ["id1", "id2", ...]})
//
// Download Endpoints
//
//	GET /files/{id}/download - Download file content
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