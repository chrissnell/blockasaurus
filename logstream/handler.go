// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

package logstream

import (
	"encoding/json"
	"net/http"

	"nhooyr.io/websocket"
)

// AcceptHook runs after a successful WebSocket upgrade and before streaming
// begins. It receives the request and the upgraded connection and returns an
// unregister function that will be invoked (via defer) when the connection
// closes. A nil unregister function is fine; callers that don't need cleanup
// can return nil.
type AcceptHook func(r *http.Request, conn *websocket.Conn) func()

// Handler returns an HTTP handler that upgrades to WebSocket and streams log entries.
func Handler(broadcaster *Broadcaster) http.HandlerFunc {
	return HandlerWithHook(broadcaster, nil)
}

// HandlerWithHook is like Handler but invokes the supplied hook after the
// WebSocket upgrade succeeds. The hook's returned unregister function is
// deferred to run when the connection closes.
func HandlerWithHook(broadcaster *Broadcaster, hook AcceptHook) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			InsecureSkipVerify: true, // Allow any origin (local network service)
		})
		if err != nil {
			return
		}
		defer conn.CloseNow() //nolint:errcheck

		if hook != nil {
			if unregister := hook(r, conn); unregister != nil {
				defer unregister()
			}
		}

		ch, cancel := broadcaster.Subscribe()
		defer cancel()

		// Derive a context that cancels when the client disconnects.
		// The read loop in CloseRead detects close frames and broken connections.
		ctx := conn.CloseRead(r.Context())

		for {
			select {
			case entry, ok := <-ch:
				if !ok {
					conn.Close(websocket.StatusGoingAway, "server shutting down") //nolint:errcheck

					return
				}

				data, err := json.Marshal(entry)
				if err != nil {
					continue
				}

				if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
					return
				}

			case <-ctx.Done():
				conn.Close(websocket.StatusNormalClosure, "client disconnected") //nolint:errcheck

				return
			}
		}
	}
}
