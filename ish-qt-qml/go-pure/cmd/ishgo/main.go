package main

import (
	"context"
	"log"
	"os"

	"gioui.org/app"

	ishapp "github.com/mostafa637/mostafa637/go-pure/internal/app"
	"github.com/mostafa637/mostafa637/go-pure/internal/platform"
)

func main() {
	shell := os.Getenv("ISH_SHELL")
	sess := platform.NewPTYSession(shell)
	go func() {
		if err := ishapp.Run(context.Background(), sess); err != nil {
			log.Printf("iSH application stopped: %v", err)
		}
		os.Exit(0)
	}()
	app.Main()
}
