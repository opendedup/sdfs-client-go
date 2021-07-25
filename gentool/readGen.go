package main

import (
	context "context"
	"flag"

	"github.com/opendedup/sdfs-client-go/utils"
)

//FileCmd Configure Volume functions for sdfscli
func ReadCmd(ctx context.Context, flagSet *flag.FlagSet) {
	flagSet.String("target", "sdfss://localhost:6442", "The target of the write")
	flagSet.String("size", "1 GB", "The size of each file to write")
	flagSet.String("post-write-cmd", "", "A command to run after the write")
	flagSet.Int("concurrent-runs", 1, "Then number of concurrent files to write to.")
	connection := utils.ParseAndConnect(flagSet)
	defer connection.CloseConnection(ctx)

}
