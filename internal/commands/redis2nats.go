package commands

import "github.com/spf13/cobra"

func redis2natsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "redis2nats",
	}
	return cmd
}
