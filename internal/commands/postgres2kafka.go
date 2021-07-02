package commands

import "github.com/spf13/cobra"

func postgres2kafkaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "postgres2kafka",
	}
	return cmd
}
