package cmd

import (
	"maps"
	"testing"

	"github.com/0xERR0R/blocky/log"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func init() {
	log.Silence()
}

func TestCmd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Command Suite")
}

// keepGlobals sets the global logger and the command settings back as they are now once
// the spec ends
func keepGlobals() {
	logger := log.Log()
	level, formatter, out, hooks := logger.GetLevel(), logger.Formatter, logger.Out, maps.Clone(logger.Hooks)
	path, dHost, dPort := configPath, dnsHost, dnsPort

	DeferCleanup(func() {
		// closes a syslog output the spec left
		log.Configure(log.DefaultConfig())

		logger.SetLevel(level)
		logger.SetFormatter(formatter)
		logger.SetOutput(out)
		logger.ReplaceHooks(hooks)

		configPath, dnsHost, dnsPort = path, dHost, dPort
	})
}
