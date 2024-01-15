/*
Copyright © 2023 Yong
*/
package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/superwhys/goutils/flags"
	"github.com/superwhys/goutils/lg"
	"github.com/superwhys/ssh-proxy/server"
)

// listmeshCmd represents the listmesh command
var listmeshCmd = &cobra.Command{
	Use:                   "ls [--all]|[mesh]",
	Short:                 "List all mesh services",
	DisableFlagsInUseLine: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		all := flags.Bool("all", false, "")
		flags.Parse()

		serviceMesh := server.NewServiceMesh()

		var showLines []string
		if all() {
			meshes, err := serviceMesh.GetAllMeshes()
			if err != nil {
				return err
			}
			for _, mesh := range meshes {
				showLines = append(showLines, mesh.Name)
			}

		} else {
			if len(args) == 0 {
				return errors.New("mesh name is required")
			}

			mesh, err := serviceMesh.GetMesh(args[0])
			if err != nil {
				return err
			}
			for _, service := range mesh.Services {
				showLines = append(showLines, fmt.Sprintf("%s \n %s", service.ServiceName, service.RemoteAddr))
			}
		}
		lg.Infof("All mesh services: \n %s", prettySlices(showLines))
		return nil
	},
}

func init() {
	meshCmd.AddCommand(listmeshCmd)
	listmeshCmd.Flags().Bool("all", false, "Include mesh not only belongs to current user")
}
