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
	readCmd := flag.NewFlagSet("read", flag.ExitOnError)
	writeCmd := flag.NewFlagSet("write", flag.ExitOnError)
	defer cancel()

	if len(os.Args) == 1 {
		fmt.Println("expected 'version','read' or 'write' subcommands")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "write":
		WriteCmd(ctx, writeCmd)
	case "file":
		ReadCmd(ctx, readCmd)
	case "version":
		fmt.Printf("Version : %s\n", Version)
		fmt.Printf("Build Date: %s\n", BuildDate)
		os.Exit(0)
	default:
		fmt.Println("expected 'version','read' or 'write' subcommands")
		os.Exit(1)

	}
}
