//go:build !windows && !plan9

package cmd

import (
	"bytes"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/0xERR0R/blocky/helpertest"
	"github.com/0xERR0R/blocky/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

// bannerMarker is a fragment of printBanner's output. Upstream matches on its
// `_/_/` ASCII art, which the rebranded banner does not contain.
const bannerMarker = "Blockasaurus v"

var _ = Describe("Serve command logging to syslog", func() {
	var (
		tmpDir     *helpertest.TmpFolder
		out        *bytes.Buffer
		syslogConn net.PacketConn
	)

	BeforeEach(func() {
		tmpDir = helpertest.NewTmpFolder("config")

		keepGlobals()
		configPath = defaultConfigPath

		out = new(bytes.Buffer)
		log.Log().SetOutput(out)
		log.Log().SetFormatter(&logrus.TextFormatter{DisableColors: true})
		log.Log().SetLevel(logrus.InfoLevel)

		var err error
		syslogConn, err = net.ListenPacket("udp", "127.0.0.1:0")
		Expect(err).Should(Succeed())
		DeferCleanup(syslogConn.Close)
	})

	// writeConfig writes a config made of lines that logs to the syslog receiver at level
	writeConfig := func(level string, lines ...string) {
		cfgFile := tmpDir.CreateStringFile("config.yml", append(lines,
			"log:",
			"  level: "+level,
			"  target: syslog",
			"  syslog:",
			"    network: udp",
			"    address: "+syslogConn.LocalAddr().String(),
		)...)

		GinkgoT().Setenv(configFileEnvVar, cfgFile.Path)
	}

	// received returns what the syslog receiver got, waiting up to wait for the first record
	received := func(wait time.Duration) string {
		var all strings.Builder

		buf := make([]byte, 64*1024)

		for {
			Expect(syslogConn.SetReadDeadline(time.Now().Add(wait))).Should(Succeed())

			n, _, err := syslogConn.ReadFrom(buf)
			if err != nil {
				return all.String()
			}

			all.Write(buf[:n])

			wait = 50 * time.Millisecond
		}
	}

	// None of these fixtures carry an `upstreams:` block: this fork rejects one,
	// which would make every config here invalid for a reason unrelated to the
	// logging behavior under test. Upstream groups live in the config store.
	It("should log the banner, then what loading the config logged, to syslog", func() {
		writeConfig("info",
			"caching:",
			"  minTime: 5",
		)

		_, err := loadServerConfig()
		Expect(err).Should(Succeed())

		msgs := received(5 * time.Second)
		Expect(strings.Count(msgs, "duration without a unit")).Should(Equal(1))
		Expect(msgs).Should(ContainSubstring(bannerMarker))
		Expect(strings.Index(msgs, bannerMarker)).Should(BeNumerically("<", strings.Index(msgs, "duration without a unit")))
		Expect(out.String()).Should(BeEmpty())
	})

	It("should log what loading the config logged at the config's level", func() {
		writeConfig("warn",
			"caching:",
			"  minTime: 5",
		)

		_, err := loadServerConfig()
		Expect(err).Should(Succeed())

		msgs := received(5 * time.Second)
		Expect(msgs).Should(ContainSubstring("duration without a unit"))
		Expect(msgs).ShouldNot(ContainSubstring(bannerMarker))
	})

	DescribeTable("should log what loading the config logged as before if the config can't be used",
		func(lines []string, expected string) {
			writeConfig("info", append([]string{"caching:", "  minTime: 5"}, lines...)...)

			_, err := loadServerConfig()
			Expect(err).Should(MatchError(ContainSubstring(expected)))

			Expect(out.String()).Should(ContainSubstring("duration without a unit"))
			Expect(out.String()).ShouldNot(ContainSubstring(bannerMarker))
			Expect(received(200 * time.Millisecond)).Should(BeEmpty())
		},
		Entry("invalid", []string{"ports:", "  dohPath: nope"}, "dohPath must start with '/'"),
		Entry("DNS listen address", []string{"ports:", "  dns: 1.2.3.4:99999"}, "can't parse DNS listen address"),
		// Upstream also has an "HTTP listen address" entry here. setListenAddresses
		// only records the DNS address in this fork — the apiHost/apiPort it fed
		// belonged to the CLI subcommands the web UI replaced — so there is no
		// config-load error to assert on for ports.http.
	)

	DescribeTable("should load the config once when serve runs", func(args []string) {
		hook := test.NewLocal(log.Log())

		// a taken HTTP port makes the server fail once it has loaded the config, before it listens
		taken, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).Should(Succeed())
		DeferCleanup(taken.Close)

		// databasePath is mandatory in this fork, and ports.dns is pinned so the
		// DNS listeners — created before the HTTP ones — do not race the default
		// privileged :53 and fail with the wrong error.
		writeConfig("info",
			"databasePath: "+tmpDir.Path+"/blocky.db",
			"caching:",
			"  minTime: 5",
			"ports:",
			"  dns: 127.0.0.1:"+strconv.Itoa(helpertest.NextFreePort()),
			"  http: 127.0.0.1:"+strconv.Itoa(taken.Addr().(*net.TCPAddr).Port),
		)

		c := NewRootCommand()
		c.SetArgs(args)
		c.SilenceErrors = true

		errChan := make(chan error, 1)
		go func() { errChan <- c.Execute() }()

		Eventually(errChan, "10s").Should(Receive(MatchError(ContainSubstring("address already in use"))))

		n := 0

		for _, entry := range hook.AllEntries() {
			if strings.Contains(entry.Message, "duration without a unit") {
				n++
			}
		}

		Expect(n).Should(Equal(1))
	},
		Entry("as the default command", []string{}),
		Entry("as the serve command", []string{newServeCommand().Name()}),
	)
})
