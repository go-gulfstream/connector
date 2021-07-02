package commands

import (
	"github.com/go-gulfstream/connector/internal/config"
	"github.com/spf13/cobra"
)

func postgres2natsCommand(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use: "postgres2nats",
	}
	return cmd
}
