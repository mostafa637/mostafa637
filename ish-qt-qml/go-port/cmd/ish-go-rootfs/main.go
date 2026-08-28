package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/mostafa637/ish-qt-qml/go-port/internal/rootfs"
)

func main() {
	archivePath := flag.String("archive", "", "path to root.tar.gz")
	basePath := flag.String("base", "", "destination fakefs base directory")
	flag.Parse()
	if *archivePath == "" || *basePath == "" {
		flag.Usage()
		os.Exit(2)
	}
	archive, err := os.Open(*archivePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open archive: %v\n", err)
		os.Exit(1)
	}
	defer archive.Close()
	if err := rootfs.Install(context.Background(), archive, *basePath); err != nil {
		fmt.Fprintf(os.Stderr, "install rootfs: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("rootfs installed at %s\n", *basePath)
}
