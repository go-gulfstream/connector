package commands

import (
	"github.com/spf13/cobra"
)

func postgres2natsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "postgres2nats",
	}
	return cmd
}
