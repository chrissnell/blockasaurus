// Copyright 2026 Chris Snell
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/0xERR0R/blocky/auth"
	"github.com/0xERR0R/blocky/configstore"
	"github.com/0xERR0R/blocky/helpertest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// writeMinimalConfig writes a config.yml with databasePath set to a tmp path
// and returns (configPath, dbPath). Subcommands reopen the store themselves,
// so we don't seed it here.
func writeMinimalConfig(tmp *helpertest.TmpFolder) (cfgPath, dbPath string) {
	dbPath = filepath.Join(tmp.Path, "auth.db")
	cfgFile := tmp.CreateStringFile("config.yml",
		"log:",
		"  level: warn",
		"databasePath: "+dbPath,
	)

	return cfgFile.Path, dbPath
}

// seedUser opens the DB directly and inserts a user. Mirrors what a prior
// `user create` invocation would have done.
func seedUser(dbPath, username, role, password string) {
	store, err := configstore.Open(dbPath)
	Expect(err).ShouldNot(HaveOccurred())

	defer store.Close()

	hash, err := auth.HashPassword(password)
	Expect(err).ShouldNot(HaveOccurred())

	Expect(store.CreateUser(&configstore.User{
		Username:     username,
		PasswordHash: hash,
		Role:         role,
	})).Should(Succeed())
}

// stubPasswordReader installs a scripted sequence of responses for readPassword
// and returns a restore func. Tests call restore via DeferCleanup.
func stubPasswordReader(responses ...string) func() {
	orig := readPassword
	idx := 0
	readPassword = func(prompt string) (string, error) {
		if idx >= len(responses) {
			return "", nil
		}

		r := responses[idx]
		idx++

		return r, nil
	}

	return func() { readPassword = orig }
}

var _ = Describe("user command", func() {
	var tmp *helpertest.TmpFolder

	BeforeEach(func() {
		tmp = helpertest.NewTmpFolder("user-cmd")
		// Reset configPath so each test picks up its own config.
		configPath = defaultConfigPath
	})

	Describe("user create", func() {
		It("creates a user with valid inputs", func() {
			cfgPath, dbPath := writeMinimalConfig(tmp)

			restore := stubPasswordReader("correcthorsebatterystaple", "correcthorsebatterystaple")
			DeferCleanup(restore)

			out := &bytes.Buffer{}

			c := NewRootCommand()
			c.SetOut(out)
			c.SetErr(out)
			c.SetArgs([]string{"--config", cfgPath, "user", "create", "--username", "Alice", "--role", "admin"})

			Expect(c.Execute()).Should(Succeed())
			Expect(out.String()).Should(ContainSubstring(`Created user "alice"`))

			// Verify persistence.
			store, err := configstore.Open(dbPath)
			Expect(err).ShouldNot(HaveOccurred())

			defer store.Close()

			u, err := store.GetUserByUsername("alice")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(u.Role).Should(Equal("admin"))
			Expect(auth.CheckPassword(u.PasswordHash, "correcthorsebatterystaple")).Should(BeTrue())
		})

		It("rejects an invalid role", func() {
			cfgPath, _ := writeMinimalConfig(tmp)

			// Password reader shouldn't be called; stub anyway for safety.
			restore := stubPasswordReader("pw1pw1pw1pw1", "pw1pw1pw1pw1")
			DeferCleanup(restore)

			c := NewRootCommand()
			c.SetOut(&bytes.Buffer{})
			c.SetErr(&bytes.Buffer{})
			c.SetArgs([]string{"--config", cfgPath, "user", "create", "--username", "bob", "--role", "superuser"})

			err := c.Execute()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("invalid role"))
		})

		It("rejects mismatched password confirmation", func() {
			cfgPath, _ := writeMinimalConfig(tmp)

			restore := stubPasswordReader("correcthorsebatterystaple", "somethingelse12")
			DeferCleanup(restore)

			c := NewRootCommand()
			c.SetOut(&bytes.Buffer{})
			c.SetErr(&bytes.Buffer{})
			c.SetArgs([]string{"--config", cfgPath, "user", "create", "--username", "bob", "--role", "viewer"})

			err := c.Execute()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("passwords do not match"))
		})

		It("rejects a password shorter than the minimum", func() {
			cfgPath, _ := writeMinimalConfig(tmp)

			restore := stubPasswordReader("short", "short")
			DeferCleanup(restore)

			c := NewRootCommand()
			c.SetOut(&bytes.Buffer{})
			c.SetErr(&bytes.Buffer{})
			c.SetArgs([]string{"--config", cfgPath, "user", "create", "--username", "bob", "--role", "viewer"})

			err := c.Execute()
			Expect(err).Should(HaveOccurred())
		})

		It("errors when required flags are missing", func() {
			cfgPath, _ := writeMinimalConfig(tmp)

			c := NewRootCommand()
			c.SetOut(&bytes.Buffer{})
			c.SetErr(&bytes.Buffer{})
			c.SetArgs([]string{"--config", cfgPath, "user", "create"})

			err := c.Execute()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(SatisfyAny(
				ContainSubstring("username"),
				ContainSubstring("role"),
			))
		})
	})

	Describe("user list", func() {
		It("prints a header and user rows", func() {
			cfgPath, dbPath := writeMinimalConfig(tmp)
			seedUser(dbPath, "alice", "admin", "correcthorsebatterystaple")
			seedUser(dbPath, "bob", "viewer", "correcthorsebatterystaple")

			out := &bytes.Buffer{}

			c := NewRootCommand()
			c.SetOut(out)
			c.SetErr(out)
			c.SetArgs([]string{"--config", cfgPath, "user", "list"})

			Expect(c.Execute()).Should(Succeed())

			s := out.String()
			Expect(s).Should(ContainSubstring("USERNAME"))
			Expect(s).Should(ContainSubstring("alice"))
			Expect(s).Should(ContainSubstring("bob"))
			Expect(s).Should(ContainSubstring("admin"))
			Expect(s).Should(ContainSubstring("viewer"))
		})

		It("prints a friendly message when there are no users", func() {
			cfgPath, _ := writeMinimalConfig(tmp)

			out := &bytes.Buffer{}

			c := NewRootCommand()
			c.SetOut(out)
			c.SetErr(out)
			c.SetArgs([]string{"--config", cfgPath, "user", "list"})

			Expect(c.Execute()).Should(Succeed())
			Expect(out.String()).Should(ContainSubstring("No users"))
		})
	})

	Describe("user delete", func() {
		It("deletes a user with --yes", func() {
			cfgPath, dbPath := writeMinimalConfig(tmp)
			seedUser(dbPath, "alice", "admin", "correcthorsebatterystaple")
			seedUser(dbPath, "bob", "viewer", "correcthorsebatterystaple")

			out := &bytes.Buffer{}

			c := NewRootCommand()
			c.SetOut(out)
			c.SetErr(out)
			c.SetArgs([]string{"--config", cfgPath, "user", "delete", "--username", "bob", "--yes"})

			Expect(c.Execute()).Should(Succeed())
			Expect(out.String()).Should(ContainSubstring(`Deleted user "bob"`))

			// Verify gone.
			store, err := configstore.Open(dbPath)
			Expect(err).ShouldNot(HaveOccurred())

			defer store.Close()

			_, err = store.GetUserByUsername("bob")
			Expect(err).Should(HaveOccurred())
		})

		It("aborts when the user answers 'n' at the confirmation prompt", func() {
			cfgPath, dbPath := writeMinimalConfig(tmp)
			seedUser(dbPath, "alice", "admin", "correcthorsebatterystaple")
			seedUser(dbPath, "bob", "viewer", "correcthorsebatterystaple")

			out := &bytes.Buffer{}

			c := NewRootCommand()
			c.SetOut(out)
			c.SetErr(out)
			c.SetIn(strings.NewReader("n\n"))
			c.SetArgs([]string{"--config", cfgPath, "user", "delete", "--username", "bob"})

			Expect(c.Execute()).Should(Succeed())
			Expect(out.String()).Should(ContainSubstring("Aborted"))

			// Verify still present.
			store, err := configstore.Open(dbPath)
			Expect(err).ShouldNot(HaveOccurred())

			defer store.Close()

			_, err = store.GetUserByUsername("bob")
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("errors on unknown username", func() {
			cfgPath, dbPath := writeMinimalConfig(tmp)
			seedUser(dbPath, "alice", "admin", "correcthorsebatterystaple")

			c := NewRootCommand()
			c.SetOut(&bytes.Buffer{})
			c.SetErr(&bytes.Buffer{})
			c.SetArgs([]string{"--config", cfgPath, "user", "delete", "--username", "nobody", "--yes"})

			err := c.Execute()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring(`"nobody"`))
		})

		It("refuses to delete the last admin", func() {
			cfgPath, dbPath := writeMinimalConfig(tmp)
			seedUser(dbPath, "alice", "admin", "correcthorsebatterystaple")

			c := NewRootCommand()
			c.SetOut(&bytes.Buffer{})
			c.SetErr(&bytes.Buffer{})
			c.SetArgs([]string{"--config", cfgPath, "user", "delete", "--username", "alice", "--yes"})

			err := c.Execute()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("last admin"))
		})

		It("errors when --username is missing", func() {
			cfgPath, _ := writeMinimalConfig(tmp)

			c := NewRootCommand()
			c.SetOut(&bytes.Buffer{})
			c.SetErr(&bytes.Buffer{})
			c.SetArgs([]string{"--config", cfgPath, "user", "delete"})

			err := c.Execute()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("username"))
		})
	})

	Describe("user reset-password", func() {
		It("updates the password and invalidates sessions", func() {
			cfgPath, dbPath := writeMinimalConfig(tmp)
			seedUser(dbPath, "alice", "admin", "originalpassword12")

			// Create a session for alice up front so we can verify it's gone.
			var aliceID uint

			{
				store, err := configstore.Open(dbPath)
				Expect(err).ShouldNot(HaveOccurred())

				u, err := store.GetUserByUsername("alice")
				Expect(err).ShouldNot(HaveOccurred())
				aliceID = u.ID

				sess, err := store.CreateSession(aliceID)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(sess.ID).ShouldNot(BeEmpty())
				store.Close()
			}

			restore := stubPasswordReader("newpasswordnew12", "newpasswordnew12")
			DeferCleanup(restore)

			out := &bytes.Buffer{}

			c := NewRootCommand()
			c.SetOut(out)
			c.SetErr(out)
			c.SetArgs([]string{"--config", cfgPath, "user", "reset-password", "--username", "alice"})

			Expect(c.Execute()).Should(Succeed())
			Expect(out.String()).Should(ContainSubstring("Password reset"))

			// Verify: new password works, old does not, sessions gone.
			store, err := configstore.Open(dbPath)
			Expect(err).ShouldNot(HaveOccurred())

			defer store.Close()

			u, err := store.GetUserByUsername("alice")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(auth.CheckPassword(u.PasswordHash, "newpasswordnew12")).Should(BeTrue())
			Expect(auth.CheckPassword(u.PasswordHash, "originalpassword12")).Should(BeFalse())
		})

		It("errors when the user does not exist", func() {
			cfgPath, _ := writeMinimalConfig(tmp)

			c := NewRootCommand()
			c.SetOut(&bytes.Buffer{})
			c.SetErr(&bytes.Buffer{})
			c.SetArgs([]string{"--config", cfgPath, "user", "reset-password", "--username", "ghost"})

			err := c.Execute()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring(`"ghost"`))
		})

		It("rejects mismatched confirmation", func() {
			cfgPath, dbPath := writeMinimalConfig(tmp)
			seedUser(dbPath, "alice", "admin", "originalpassword12")

			restore := stubPasswordReader("newpasswordnew12", "somethingelse123")
			DeferCleanup(restore)

			c := NewRootCommand()
			c.SetOut(&bytes.Buffer{})
			c.SetErr(&bytes.Buffer{})
			c.SetArgs([]string{"--config", cfgPath, "user", "reset-password", "--username", "alice"})

			err := c.Execute()
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("passwords do not match"))
		})

		It("rejects a too-short password", func() {
			cfgPath, dbPath := writeMinimalConfig(tmp)
			seedUser(dbPath, "alice", "admin", "originalpassword12")

			restore := stubPasswordReader("short", "short")
			DeferCleanup(restore)

			c := NewRootCommand()
			c.SetOut(&bytes.Buffer{})
			c.SetErr(&bytes.Buffer{})
			c.SetArgs([]string{"--config", cfgPath, "user", "reset-password", "--username", "alice"})

			err := c.Execute()
			Expect(err).Should(HaveOccurred())
		})
	})
})
