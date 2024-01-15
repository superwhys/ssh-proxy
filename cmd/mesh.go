/*
Copyright © 2023 Yong

*/
package cmd

import (
	"github.com/spf13/cobra"
)

// meshCmd represents the mesh command
var meshCmd = &cobra.Command{
	Use: "mesh",
}

func init() {
	rootCmd.AddCommand(meshCmd)
}
