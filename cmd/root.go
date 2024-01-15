/*
Copyright © 2023 Yong
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/superwhys/goutils/flags"
	"github.com/superwhys/sshtunnel"
)

var (
	env            = flags.String("env", "dev", "Environment name for looking up connection profile")
	profiles       = flags.Struct("profiles", []*ConnectionProfile{}, "Connection profiles")
	privateKeyPath = flags.String("privateKey", os.Getenv("HOME")+"/.ssh/id_rsa", "private key")
	port           = flags.Int("port", 0, "Port for serivce")
)

type ConnectionProfile struct {
	EnvName string
	Hosts   []*sshtunnel.SshConfig
}

func (cp *ConnectionProfile) PopulateDefault(identityFile string) {
	for _, h := range cp.Hosts {
		if h.IdentityFile == "" {
			h.IdentityFile = identityFile
		}
		if h.User == "" {
			h.User = "root"
		}
	}
}

var rootCmd = &cobra.Command{
	Use:   "ssh-proxy",
	Short: "Handy command line tool for connecting to remote services.",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	flags.OverrideDefaultConfigFile(os.Getenv("HOME") + "/.ssh-proxy.yaml")
}
