package main

import (
	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:   "polyrepo",
	Short: "Get a wrangle on your tests and schedule like a boss 🚀.",
}

func main() {
	root.PersistentFlags().StringP("config", "c", "", "the path to the polyrepo config file")
	root.Execute()
}
