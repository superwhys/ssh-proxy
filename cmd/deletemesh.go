/*
Copyright © 2023 Yong
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/superwhys/goutils/lg"
	"github.com/superwhys/ssh-proxy/server"
)

// deletemeshCmd represents the deletemesh command
var deletemeshCmd = &cobra.Command{
	Use:   "delete [mesh1] [mesh2] ...",
	Short: "Delete mesh",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		serviceMesh := server.NewServiceMesh()
		for _, mesh := range args {
			if err := serviceMesh.DeleteMesh(mesh); err != nil {
				return err
			}
			lg.Infof("Mesh %s deleted", mesh)
		}

		return nil
	},
}

func init() {
	meshCmd.AddCommand(deletemeshCmd)
}
