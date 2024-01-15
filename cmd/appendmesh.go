/*
Copyright © 2023 Yong
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/superwhys/goutils/flags"
	"github.com/superwhys/ssh-proxy/server"
)

// appendCmd represents the append command
var appendCmd = &cobra.Command{
	Use:   "append [mesh] [service1] [service2] ...",
	Short: "Append services to existing mesh",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		flags.Parse()
		meshName := args[0]
		services, err := parseHostPortPairs(args[1:]...)
		if err != nil {
			return err
		}

		serviceMesh := server.NewServiceMesh()

		for _, service := range services {
			if err := serviceMesh.AddServiceToMesh(meshName, &server.Service{
				RemoteAddr:  service.RemoteAddress,
				ServiceName: service.ServiceName,
			}); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	meshCmd.AddCommand(appendCmd)
}
