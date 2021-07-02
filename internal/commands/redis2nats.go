package commands

import (
	"github.com/go-gulfstream/connector/internal/config"
	"github.com/spf13/cobra"
)

func redis2natsCommand(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use: "redis2nats",
	}
	return cmd
}
