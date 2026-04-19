// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

// Package auth provides authentication primitives: password hashing,
// session token generation, and role validation. It is intentionally
// dependency-free — no storage, no HTTP — so it can be imported by
// both the HTTP middleware and the CLI user-management commands.
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinPasswordLength is the minimum acceptable password length in bytes.
	MinPasswordLength = 12

	// SessionDuration is the lifetime of a new or renewed session.
	SessionDuration = 24 * time.Hour

	// SessionSlidingThreshold is the remaining-time threshold below which
	// an active session is extended by SessionDuration.
	SessionSlidingThreshold = 12 * time.Hour

	// SessionCookieName is the session cookie name used over plain HTTP.
	SessionCookieName = "blockasaurus_session"

	// SessionCookieNameSecure is the session cookie name used over HTTPS.
	// The __Host- prefix is a browser-enforced invariant: the cookie must
	// have Secure, Path=/, and no Domain attribute, which prevents
	// subdomain-scoped cookie fixation. See Phase 4 for the rationale.
	SessionCookieNameSecure = "__Host-blockasaurus_session"

	// SessionTokenBytes is the number of random bytes read from crypto/rand
	// when generating a session token. The hex-encoded form is twice this
	// length (64 characters for 32 bytes).
	SessionTokenBytes = 32

	// RoleAdmin grants full read/write access.
	RoleAdmin = "admin"

	// RoleViewer grants read-only access.
	RoleViewer = "viewer"
)

// ErrEmptyPassword is returned by HashPassword when given an empty password.
var ErrEmptyPassword = errors.New("password must not be empty")

// HashPassword returns a bcrypt hash of the password at bcrypt.DefaultCost.
// An empty password is rejected up-front since bcrypt would otherwise
// silently accept it.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	return string(hash), nil
}

// CheckPassword reports whether password matches the bcrypt hash.
// Any bcrypt error (mismatch, malformed hash, etc.) is treated as false;
// callers cannot distinguish "wrong password" from "bad hash" — both mean
// authentication fails.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// ValidatePassword enforces the minimum password length policy.
// It does not hash or store anything.
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}

	return nil
}

// GenerateSessionToken returns a hex-encoded cryptographically random token
// suitable for use as an opaque session identifier. The returned string is
// 2 * SessionTokenBytes characters long (64 chars for the default 32 bytes).
func GenerateSessionToken() (string, error) {
	buf := make([]byte, SessionTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}

	return hex.EncodeToString(buf), nil
}

// ValidateRole accepts exactly RoleAdmin or RoleViewer and rejects anything
// else, including case variants and surrounding whitespace.
func ValidateRole(role string) error {
	switch role {
	case RoleAdmin, RoleViewer:
		return nil
	default:
		return fmt.Errorf("invalid role %q: must be %q or %q", role, RoleAdmin, RoleViewer)
	}
}

// UsernameMinLength is the minimum acceptable username length.
const UsernameMinLength = 3

// UsernameMaxLength is the maximum acceptable username length.
const UsernameMaxLength = 64

// ValidateUsername enforces the username policy: 3-64 characters drawn from
// [a-z0-9._-]. Callers MUST lowercase the input first; this function rejects
// any uppercase character rather than silently accepting it.
func ValidateUsername(u string) error {
	if len(u) < UsernameMinLength || len(u) > UsernameMaxLength {
		return fmt.Errorf("username must be %d-%d characters, lowercase alphanumerics, dots, underscores, or hyphens",
			UsernameMinLength, UsernameMaxLength)
	}

	for _, r := range u {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-':
		default:
			return fmt.Errorf("username must be %d-%d characters, lowercase alphanumerics, dots, underscores, or hyphens",
				UsernameMinLength, UsernameMaxLength)
		}
	}

	return nil
}
