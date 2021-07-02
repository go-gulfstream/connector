package commands

import "github.com/spf13/cobra"

func New() *cobra.Command {
	root := &cobra.Command{
		Use: "gc-connector",
	}
	root.AddCommand(postgres2kafkaCommand())
	root.AddCommand(postgres2natsCommand())
	root.AddCommand(redis2kafkaCommand())
	root.AddCommand(redis2natsCommand())
	return root
}
