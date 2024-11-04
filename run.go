package main

import (
	"log"

	"github.com/spf13/cobra"
)

func init() {
	root.AddCommand(runCommand)
}

var runCommand = &cobra.Command{
	Use:   "run",
	Short: "run the commands for each repository in the workspace",
	Long:  "run the commands for each repository in the workspace",
	Run: func(cmd *cobra.Command, args []string) {
		config := cmd.Flag("config").Value.String()
		log.Printf("config: %s", config)
	},
}
