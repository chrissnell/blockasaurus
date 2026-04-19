// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

// WSRevoker tracks authenticated WebSocket connections keyed by the owning
// user ID. It does two things:
//
//  1. Per-connection lifetime cap: a goroutine closes the connection at the
//     session's ExpiresAt instant with close code 1008 Policy Violation.
//     Sliding-renewed sessions do not extend an already-open socket — the
//     client must reconnect to benefit from a renewed expiry, which
//     re-runs RequireAuth at the upgrade.
//  2. Revocation on demand: consumes store.SessionRevoked() and closes every
//     registered connection for the revoked user.
//
// The zero value is not usable; construct with NewWSRevoker. A single
// long-lived revoker per server process is the expected pattern.
type WSRevoker struct {
	mu    sync.Mutex
	conns map[uint]map[*websocket.Conn]struct{}
}

// NewWSRevoker constructs an empty revoker. Call StartRevoker to begin
// consuming revocation events.
func NewWSRevoker() *WSRevoker {
	return &WSRevoker{
		conns: make(map[uint]map[*websocket.Conn]struct{}),
	}
}

// Register tracks conn under userID and starts a lifetime-cap goroutine
// that closes the connection at expiresAt. Returns an unregister closure
// that MUST be invoked by the handler when the connection ends for any
// reason — on client disconnect, handler return, or normal shutdown — so
// the revoker doesn't leak tracking entries or stray timers.
//
// The returned closure is idempotent; calling it more than once is a
// no-op.
func (r *WSRevoker) Register(userID uint, conn *websocket.Conn, expiresAt time.Time) func() {
	r.mu.Lock()

	set, ok := r.conns[userID]
	if !ok {
		set = make(map[*websocket.Conn]struct{})
		r.conns[userID] = set
	}

	set[conn] = struct{}{}
	r.mu.Unlock()

	done := make(chan struct{})

	// Lifetime cap: close at expiresAt. Cancelled via done when the
	// handler unregisters first.
	go func() {
		d := time.Until(expiresAt)
		if d <= 0 {
			r.closeOne(conn)

			return
		}

		t := time.NewTimer(d)
		defer t.Stop()

		select {
		case <-t.C:
			r.closeOne(conn)
		case <-done:
		}
	}()

	var once sync.Once

	return func() {
		once.Do(func() {
			close(done)

			r.mu.Lock()
			defer r.mu.Unlock()

			if set, ok := r.conns[userID]; ok {
				delete(set, conn)
				if len(set) == 0 {
					delete(r.conns, userID)
				}
			}
		})
	}
}

// closeOne closes a single connection with 1008 Policy Violation and
// drops it from the tracking map. Errors are discarded — the connection
// may already be closed from the client side.
func (r *WSRevoker) closeOne(conn *websocket.Conn) {
	_ = conn.Close(websocket.StatusPolicyViolation, "session revoked")

	r.mu.Lock()
	defer r.mu.Unlock()

	for uid, set := range r.conns {
		if _, ok := set[conn]; ok {
			delete(set, conn)
			if len(set) == 0 {
				delete(r.conns, uid)
			}

			return
		}
	}
}

// RevokeUser closes every tracked connection belonging to userID. Safe
// to call with no registered connections.
func (r *WSRevoker) RevokeUser(userID uint) {
	r.mu.Lock()
	set, ok := r.conns[userID]
	if !ok {
		r.mu.Unlock()

		return
	}

	// Snapshot then release the lock before calling Close — Close can
	// block on network flush, and closeOne re-acquires the lock.
	victims := make([]*websocket.Conn, 0, len(set))
	for c := range set {
		victims = append(victims, c)
	}

	delete(r.conns, userID)
	r.mu.Unlock()

	for _, c := range victims {
		_ = c.Close(websocket.StatusPolicyViolation, "session revoked")
	}
}

// StartRevoker consumes store.SessionRevoked() and calls RevokeUser for
// every emitted userID. Runs until ctx is cancelled. Intended to be
// started once from the server bootstrap.
func StartRevoker(ctx context.Context, store SessionStore, revoker *WSRevoker) {
	go func() {
		ch := store.SessionRevoked()

		for {
			select {
			case <-ctx.Done():
				return
			case uid, ok := <-ch:
				if !ok {
					return
				}

				revoker.RevokeUser(uid)
			}
		}
	}()
}
