package commands

import (
	"github.com/spf13/cobra"
)

func redis2kafkaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "redis2kafka",
	}
	return cmd
}
