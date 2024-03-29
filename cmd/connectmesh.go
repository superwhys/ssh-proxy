/*
Copyright © 2023 Yong
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/superwhys/goutils/flags"
	"github.com/superwhys/goutils/lg"
	"github.com/superwhys/ssh-proxy/server"
	"github.com/superwhys/ssh-proxy/sshproxypb"
)

// connectmeshCmd represents the connectmesh command
var connectmeshCmd = &cobra.Command{
	Use:   "connect [mesh]",
	Short: "Build tunnel to set of services",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		flags.Parse()

		meshName := args[0]
		meshUtils := server.NewServiceMesh()

		mesh, err := meshUtils.GetMesh(meshName)
		if err != nil {
			lg.Errorf("Failed to get mesh %s: %v", meshName, err)
			return err
		}

		proxyHosts := make([]*sshproxypb.Service, 0, len(mesh.Services))
		for _, service := range mesh.Services {
			proxyHosts = append(proxyHosts, &sshproxypb.Service{
				ServiceName:  service.ServiceName,
				ProxyAddress: service.RemoteAddr,
			})
		}

		if len(proxyHosts) == 0 {
			lg.Errorf("No services found in mesh %s", meshName)
			return nil
		}

		env = func() string {
			return mesh.Env
		}
		lg.Infof("Starting connect to mesh %s, env %s", meshName, env())

		err = startConnect(proxyHosts)
		if err != nil {
			lg.Errorf("Failed to start connect: %v", err)
			os.Exit(1)
		}
		return nil
	},
}

func init() {
	meshCmd.AddCommand(connectmeshCmd)
}
