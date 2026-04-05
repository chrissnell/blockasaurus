package cmd

import (
	"github.com/0xERR0R/blocky/helpertest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validate command", func() {
	var tmpDir *helpertest.TmpFolder
	BeforeEach(func() {
		tmpDir = helpertest.NewTmpFolder("config")
	})
	When("Validate is called with not existing configuration file", func() {
		It("should terminate with error", func() {
			c := NewRootCommand()
			c.SetArgs([]string{"validate", "--config", "/notexisting/path.yaml"})

			Expect(c.Execute()).Should(HaveOccurred())
		})
	})

	When("Validate is called with existing valid configuration file", func() {
		It("should terminate without error", func() {
			cfgFile := tmpDir.CreateStringFile("config.yaml",
				"log:",
				"  level: warn")

			c := NewRootCommand()
			c.SetArgs([]string{"validate", "--config", cfgFile.Path})

			Expect(c.Execute()).Should(Succeed())
		})
	})

	When("Validate is called with existing invalid configuration file", func() {
		It("should terminate with error", func() {
			cfgFile := tmpDir.CreateStringFile("config.yaml",
				"conditional:",
				"  mapping:",
				"    bogus: 1.broken file")

			c := NewRootCommand()
			c.SetArgs([]string{"validate", "--config", cfgFile.Path})

			Expect(c.Execute()).Should(HaveOccurred())
		})
	})
})
