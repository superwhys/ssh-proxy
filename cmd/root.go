/*
Copyright © 2023 Yong
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/superwhys/goutils/flags"
	"github.com/superwhys/goutils/lg"
	"github.com/superwhys/sshtunnel"
)

var (
	env            = flags.String("env", "", "Environment name for looking up connection profile")
	profiles       = flags.Struct("profiles", []*ConnectionProfile{}, "Connection profiles")
	privateKeyPath = flags.String("privateKey", os.Getenv("HOME")+"/.ssh/id_rsa", "private key")
	port           = flags.Int("port", 0, "Port for serivce")

	debug bool
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
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if debug {
			lg.EnableDebug()
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	flags.OverrideDefaultConfigFile(os.Getenv("HOME") + "/.ssh-proxy.yaml")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "debug mode")
}
