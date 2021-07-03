package main

import (
	"fmt"
	"os"

	"github.com/go-gulfstream/connector/internal/commands"
)

func main() {
	app, err := commands.New()
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
	if err := app.Execute(); err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
}
