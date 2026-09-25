package cmd

import "github.com/spf13/cobra"

var stopCmd = &cobra.Command{
	Use:   "stop [flags]",
	Short: "Stops running task",
}
