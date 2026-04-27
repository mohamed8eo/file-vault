// Package middleware provides HTTP middleware utilities for request
// authentication and rate limiting in the file-vault service.
//
// It includes:
//
//   - Auth middleware:
//     Validates JWT access tokens from the Authorization header using
//     a Bearer scheme. On success, it injects the authenticated user's
//     ID into the request context for downstream handlers.
//
//   - RateLimit middleware:
//     Implements request rate limiting using Redis. It tracks request
//     counts within a configurable time window and enforces limits
//     based on either the authenticated user ID (if available) or the
//     client IP address.
//
// Context Values:
//
//   - UserIDKey:
//     A context key used to store and retrieve the authenticated user's ID
//     from the request context.
//
// This package is designed to be composable and used with standard
// net/http handlers.
package middleware
