// Package client provides a CLI client for the file-vault API.
//
// This package enables command-line interaction with the file-vault backend,
// handling authentication, token management, and file operations.
//
// Main Components
//
//   - client.go: Core client functionality
//     * Login: Authenticate user and store tokens
//     * Register: Register new user
//     * Logout: Clear stored tokens
//     * loadConfig/saveTokens: Token persistence in ~/.file-vault/config.json
//     * AuthRequest: HTTP client with automatic token refresh
//     * refreshAccessToken: Handle 401 responses by refreshing tokens
//
//   - files.go: File operation commands
//     * UploadFile: Upload file to server (auto-refreshes token on 401)
//     * ListFiles: List all user files with formatted output
//     * GetFile: Get file URL (opens in browser)
//     * DeleteFile: Delete a file by ID
//
// Token Management
//
// Tokens are stored in: ~/.file-vault/config.json
//
// On 401 (unauthorized) responses, the client automatically:
//   1. Calls /auth/refresh with the refresh token
//   2. Saves new tokens to config
//   3. Retries the original request
//
// Output Formatting
//
// ListFiles outputs a formatted table with:
//   - id: File UUID
//   - name: Original filename (truncated to 25 chars)
//   - created_at: Human-readable timestamp (e.g., "Apr 29, 2026 at 2:14 PM")
package client