package cmd

import (
	"os"

	"github.com/0xERR0R/blocky/config"
	"github.com/0xERR0R/blocky/log"
	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals
var configPath string

const (
	defaultConfigPath   = "./config.yml"
	configFileEnvVar    = "BLOCKY_CONFIG_FILE"
	configFileEnvVarOld = "CONFIG_FILE"
)

// NewRootCommand creates a new root cli command instance
func NewRootCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "blocky",
		Short: "blocky is a DNS proxy ",
		Long: `A fast and configurable DNS Proxy
and ad-blocker for local network.

Complete documentation is available at https://github.com/0xERR0R/blocky`,
		PreRunE: initConfigPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			return newServeCommand().RunE(cmd, args)
		},
		SilenceUsage: true,
	}

	c.PersistentFlags().StringVarP(&configPath, "config", "c", defaultConfigPath, "path to config file or folder")

	c.AddCommand(
		NewVersionCommand(),
		newServeCommand(),
		NewHealthcheckCommand(),
		NewValidateCommand(),
		newUserCommand(),
	)

	return c
}

func initConfigPreRun(cmd *cobra.Command, args []string) error {
	return initConfig()
}

func initConfig() error {
	if configPath == defaultConfigPath {
		val, present := os.LookupEnv(configFileEnvVar)
		if present {
			configPath = val
		} else {
			val, present = os.LookupEnv(configFileEnvVarOld)
			if present {
				configPath = val
			}
		}
	}

	cfg, err := config.LoadConfig(configPath, false)
	if err != nil {
		return err
	}

	log.Configure(&cfg.Log)

	return nil
}

// Execute starts the command
func Execute() {
	if err := NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
