// Package handler provides HTTP handlers for the file-vault service,
// including user authentication, session management, and file operations.
//
// It includes:
//
//   - Authentication handlers:
//     Handles user registration (SignUp), login (Login), logout (Logout),
//     and token refresh (Refresh). It uses JWT for access and refresh tokens,
//     stores refresh tokens in the database, and manages secure HTTP-only cookies.
//
//   - File handlers:
//     Supports uploading, retrieving, and deleting user files. Files are stored
//     in AWS S3, while metadata is persisted in the database. File access is
//     restricted to authenticated users.
//
// Authentication & Security:
//
//   - Access tokens are short-lived and returned in responses.
//   - Refresh tokens are long-lived, stored in the database, and set as HTTP-only cookies.
//   - Middleware is used to inject authenticated user identity into request context.
//
// Storage:
//
//   - File contents are stored in S3.
//   - File metadata and refresh tokens are stored in the database.
//
// This package is designed to work with net/http and integrates with the
// middleware package for authentication and request context propagation.
package handler
