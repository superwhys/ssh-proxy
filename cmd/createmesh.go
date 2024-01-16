/*
Copyright © 2023 Yong
*/
package cmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/superwhys/goutils/flags"
	"github.com/superwhys/ssh-proxy/server"
)

// createmeshCmd represents the createmesh command
var createmeshCmd = &cobra.Command{
	Use:   "create --env dev [mesh] [service1] [service2] ...",
	Short: "Create a mesh of multiple services",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		flags.Parse()
		meshName := args[0]

		if env() == "" {
			return errors.New("no env provide")
		}

		services, err := parseProfileHostPort(args[1:]...)
		if err != nil {
			return err
		}

		serviceMesh := server.NewServiceMesh()
		mesh := server.Mesh{
			Name: meshName,
			Env:  env(),
		}
		for _, service := range services {
			mesh.Services = append(mesh.Services, server.Service{RemoteAddr: service.ProxyAddress, ServiceName: service.ServiceName})
		}

		return serviceMesh.CreateMesh(mesh)
	},
}

func init() {
	meshCmd.AddCommand(createmeshCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createmeshCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createmeshCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
