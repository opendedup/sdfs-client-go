package main

import (
	context "context"
	"flag"
	"fmt"
	"os"
)

var Version = "development"
var BuildDate = "NAN"

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	configCmd := flag.NewFlagSet("config", flag.ExitOnError)
	fileCmd := flag.NewFlagSet("file", flag.ExitOnError)
	cloudCmd := flag.NewFlagSet("cloud", flag.ExitOnError)

	/*
		flag.Parse()
		if len(os.Args) == 1 {
			fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
			flag.PrintDefaults()
			os.Exit(0)

		}
	*/
	if len(os.Args) == 1 {
		fmt.Println("expected 'config','file', or 'cloud' subcommands")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "config":
		ConfigCmd(ctx, configCmd)
	case "file":
		FileCmd(ctx, fileCmd)
	case "cloud":
		CloudCmd(ctx, cloudCmd)
	default:
		fmt.Println("expected 'config','file', or 'cloud' subcommands")
		os.Exit(1)

	}

}
