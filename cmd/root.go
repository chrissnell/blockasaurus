package cmd

import (
	"fmt"
	"net"
	"os"

	"github.com/0xERR0R/blocky/config"
	"github.com/0xERR0R/blocky/log"
	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals
var (
	configPath string
	dnsHost    = defaultIPAddress
	dnsPort    = uint16(defaultDNSPort)
)

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

	if len(cfg.Ports.DNS) != 0 {
		host, port, err := splitListenAddress(cfg.Ports.DNS[0])
		if err != nil {
			return fmt.Errorf("can't parse DNS listen address '%s': %w", cfg.Ports.DNS[0], err)
		}

		if host != "" {
			if ip := net.ParseIP(host); ip == nil || !ip.IsUnspecified() {
				dnsHost = host
			}
		}

		dnsPort = port
	}

	return nil
}

// splitListenAddress splits a `ports.*` entry into host and port. Entries are normalized
// to ":port" when no host is given, so an empty host means "all interfaces".
func splitListenAddress(addr string) (host string, port uint16, err error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		// Not a host:port pair - treat the whole value as a port.
		host, portStr = "", addr
	}

	port, err = config.ConvertPort(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("can't convert port '%s' to number (1 - 65535): %w", portStr, err)
	}

	return host, port, nil
}

// Execute starts the command
func Execute() {
	if err := NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
