// Package auth provides authentication utilities for the file-vault service.
//
// This package contains two main components:
//
//   - jwt.go: JWT token handling
//     * GenerateAccessToken: Create signed JWT access tokens
//     * GenerateRefreshToken: Create signed JWT refresh tokens
//     * ValidateAccessToken: Verify and parse access tokens
//     * ValidateRefreshToken: Verify and parse refresh tokens
//     * GetUserIDFromToken: Extract user ID from token claims
//
//   - password.go: Password hashing and verification
//     * HashPassword: Hash password using Argon2id
//     * CheckPassword: Verify password against hash
//
// Security Notes
//
//   - Access tokens: Short-lived (15 minutes), contain user ID
//   - Refresh tokens: Long-lived (30 days), contain user ID
//   - Passwords: Argon2id with automatic salt generation
package auth