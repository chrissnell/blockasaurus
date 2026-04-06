//go:build !windows

package winservice

import "context"

// IsService always returns false on non-Windows platforms.
func IsService() bool { return false }

// RunFunc is the server's main blocking function.
type RunFunc func(ctx context.Context) error

// Run is a no-op on non-Windows platforms.
func Run(_ RunFunc) error { return nil }
