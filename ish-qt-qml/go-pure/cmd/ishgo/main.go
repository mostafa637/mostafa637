package main

import (
	"context"
	"log"
	"os"

	"gioui.org/app"

	ishapp "github.com/mostafa637/mostafa637/go-pure/internal/app"
	"github.com/mostafa637/mostafa637/go-pure/internal/core"
	coresession "github.com/mostafa637/mostafa637/go-pure/internal/core/session"
	"github.com/mostafa637/mostafa637/go-pure/internal/platform"
	transport "github.com/mostafa637/mostafa637/go-pure/internal/session"
)

func main() {
	ctx := context.Background()
	factory := newSessionFactory(ctx)
	go func() {
		if err := ishapp.RunManaged(ctx, factory); err != nil {
			log.Printf("iSH application stopped: %v", err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func newSessionFactory(ctx context.Context) transport.Factory {
	return func() transport.Session {
		sess, err := newSession(ctx)
		if err != nil {
			log.Printf("create session: %v", err)
			return nil
		}
		return sess
	}
}

func newSession(ctx context.Context) (transport.Session, error) {
	rootfs := os.Getenv("ISH_ROOTFS")
	if rootfs == "" {
		// Host PTY remains useful for Linux renderer development when a rootfs has
		// not been supplied. Production/rootfs runs use the GoFactory below.
		return platform.NewPTYSession(os.Getenv("ISH_SHELL")), nil
	}
	factory := core.GoFactory{Config: coresession.Config{
		Rootfs:    rootfs,
		MetaDB:    os.Getenv("ISH_META_DB"),
		Shell:     os.Getenv("ISH_SHELL"),
		GuestELF:  os.Getenv("ISH_GUEST_ELF"),
		UseGuest:  true,
		UID:       0,
		GID:       0,
		Bootstrap: true,
	}}
	return factory.New(ctx)
}
