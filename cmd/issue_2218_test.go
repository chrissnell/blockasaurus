package cmd

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/0xERR0R/blocky/helpertest"
	"github.com/0xERR0R/blocky/log"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Reproduces #2218: the Docker image runs `blocky healthcheck` with no flags, but the
// command defaults to port 53 and never looks at `ports.dns`, so a container that
// listens on a different DNS port is reported unhealthy.
var _ = Describe("Healthcheck derives the port from the config (#2218)", func() {
	var cfgFile string

	writeConfig := func(body string) {
		dir := GinkgoT().TempDir()
		cfgFile = filepath.Join(dir, "config.yml")
		Expect(os.WriteFile(cfgFile, []byte(body), 0o600)).Should(Succeed())

		GinkgoT().Setenv("BLOCKY_CONFIG_FILE", cfgFile)

		// initConfig() only consults the env var while configPath is still the default.
		old := configPath
		configPath = defaultConfigPath
		DeferCleanup(func() { configPath = old })
	}

	It("probes the DNS port from ports.dns when no flag is given", func() {
		ip := "127.0.0.1"
		p := helpertest.NextFreePort()
		port := strconv.Itoa(p)
		hostPort := helpertest.HostPort(ip, p)

		startMockServer(hostPort)

		writeConfig("ports:\n  dns: " + port + "\n")

		Eventually(func() error {
			c := NewHealthcheckCommand()
			c.SetArgs([]string{})

			return c.Execute()
		}, "2s").Should(Succeed())
	})

	It("still lets an explicit --port win over the config", func() {
		ip := "127.0.0.1"
		p := helpertest.NextFreePort()
		port := strconv.Itoa(p)
		hostPort := helpertest.HostPort(ip, p)

		startMockServer(hostPort)

		// Config points somewhere nothing is listening; the flag must take precedence.
		writeConfig("ports:\n  dns: " + strconv.Itoa(helpertest.NextFreePort()) + "\n")

		Eventually(func() error {
			c := NewHealthcheckCommand()
			c.SetArgs([]string{"-p", port, "-b", ip})

			return c.Execute()
		}, "2s").Should(Succeed())
	})

	It("keeps working when no config file exists at all", func() {
		ip := "127.0.0.1"
		p := helpertest.NextFreePort()
		port := strconv.Itoa(p)
		hostPort := helpertest.HostPort(ip, p)

		startMockServer(hostPort)

		GinkgoT().Setenv("BLOCKY_CONFIG_FILE", filepath.Join(GinkgoT().TempDir(), "does-not-exist.yml"))
		old := configPath
		configPath = defaultConfigPath
		DeferCleanup(func() { configPath = old })

		Eventually(func() error {
			c := NewHealthcheckCommand()
			c.SetArgs([]string{"-p", port, "-b", ip})

			return c.Execute()
		}, "2s").Should(Succeed())
	})

	It("keeps working and logs why when the config is invalid", func() {
		keepGlobals()
		hook := test.NewLocal(log.Log())

		ip := "127.0.0.1"
		p := helpertest.NextFreePort()
		port := strconv.Itoa(p)
		hostPort := helpertest.HostPort(ip, p)

		startMockServer(hostPort)

		// No `upstreams:` block: this fork rejects one outright, which would make
		// the config invalid for a different reason than the one under test.
		writeConfig("rateLimit:\n  enable: true\n")

		Eventually(func() error {
			c := NewHealthcheckCommand()
			c.SetArgs([]string{"-p", port, "-b", ip})

			return c.Execute()
		}, "2s").Should(Succeed())

		Expect(hook.AllEntries()).Should(ContainElement(SatisfyAll(
			HaveField("Level", logrus.WarnLevel),
			HaveField("Message", ContainSubstring("rate must be > 0 when enabled")),
		)))
	})
})
