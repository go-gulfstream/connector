package commands

import (
	"github.com/go-gulfstream/connector/internal/config"
	"github.com/spf13/cobra"
)

func postgres2kafkaCommand(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:        "postgres2kafka",
		Aliases:    []string{"p2k", "pg2kfk"},
		SuggestFor: []string{"p2", "post", "postgres", "pg"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPostgres2kafkaCommand()
		},
	}
	return cmd
}

func runPostgres2kafkaCommand() error {
	return nil
}
