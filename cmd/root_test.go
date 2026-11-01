package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/0xERR0R/blocky/log"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/0xERR0R/blocky/helpertest"
)

var _ = Describe("root command", func() {
	When("Version command is called", func() {
		log.Log().ExitFunc = nil
		It("should execute without error", func() {
			c := NewRootCommand()
			c.SetOut(io.Discard)
			c.SetArgs([]string{"help"})
			err := c.Execute()
			Expect(err).Should(Succeed())
		})
	})

	When("Config provided", func() {
		var (
			tmpDir  *TmpFolder
			tmpFile *TmpFile
		)

		BeforeEach(func() {
			configPath = defaultConfigPath
		})

		It("should accept old env var", func() {
			tmpDir = NewTmpFolder("RootCommand")
			tmpFile = tmpDir.CreateStringFile("config",
				"blocking:",
				"  denylists:",
				"    ads:",
				"      - https://s3.amazonaws.com/lists.disconnect.me/simple_ad.txt",
				"  clientGroupsBlock:",
				"    default:",
				"      - ads",
				"port: 5333",
			)

			os.Setenv(configFileEnvVarOld, tmpFile.Path)
			DeferCleanup(func() { os.Unsetenv(configFileEnvVarOld) })

			Expect(initConfig()).Should(Succeed())
			Expect(configPath).Should(Equal(tmpFile.Path))
		})

		It("should accept new env var", func() {
			tmpDir = NewTmpFolder("RootCommand")
			tmpFile = tmpDir.CreateStringFile("config",
				"blocking:",
				"  denylists:",
				"    ads:",
				"      - https://s3.amazonaws.com/lists.disconnect.me/simple_ad.txt",
				"  clientGroupsBlock:",
				"    default:",
				"      - ads",
				"port: 5333",
			)

			os.Setenv(configFileEnvVar, tmpFile.Path)
			DeferCleanup(func() { os.Unsetenv(configFileEnvVar) })

			Expect(initConfig()).Should(Succeed())
			Expect(configPath).Should(Equal(tmpFile.Path))
		})

		// Upstream's companion specs for this block cover apiHost/apiPort, apiURL
		// and printOkOrError, all of which belong to the API-driven subcommands we
		// replaced with the web UI. The syslog one is kept: it covers
		// commandLogConfig, which this fork does have.
		It("should keep command output in the terminal when the server logs to syslog", func() {
			keepGlobals()

			tmpDir = NewTmpFolder("RootCommand")
			syslogConfig := tmpDir.CreateStringFile("config_with_syslog",
				"log:",
				"  target: syslog",
				"  syslog:",
				"    network: udp",
				"    address: 127.0.0.1:1",
			)

			configPath = syslogConfig.Path

			Expect(initConfig()).Should(Succeed())
			Expect(fmt.Sprintf("%T", log.Log().Out)).ShouldNot(ContainSubstring("syslog"))
		})
	})

	Describe("Command execution", func() {
		BeforeEach(func() {
			configPath = defaultConfigPath
		})

		It("should create root command with all subcommands", func() {
			cmd := NewRootCommand()

			// Check if all subcommands are added
			subCmdNames := make([]string, 0, len(cmd.Commands()))
			for _, subCmd := range cmd.Commands() {
				subCmdNames = append(subCmdNames, subCmd.Name())
			}

			expectedCmds := []string{
				"version", "serve", "healthcheck", "validate", "user",
			}
			for _, expected := range expectedCmds {
				Expect(subCmdNames).Should(ContainElement(expected))
			}
		})

		It("should set flags correctly", func() {
			cmd := NewRootCommand()

			configFlag := cmd.PersistentFlags().Lookup("config")
			Expect(configFlag).ShouldNot(BeNil())
			Expect(configFlag.Shorthand).Should(Equal("c"))
			Expect(configFlag.DefValue).Should(Equal(defaultConfigPath))
		})
	})
})
