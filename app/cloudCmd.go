package main

import (
	context "context"
	"flag"
	"fmt"
	"os"
)

//CloudCmd cloud configuration functions for sdfscli
func CloudCmd(ctx context.Context, flagSet *flag.FlagSet) {
	scv := flagSet.Bool("check-consistency", false, "Syncs the Volume with the metadata stored"+
		" in the cloud to make sure both sides are consistent.")
	vcv := flagSet.Int64("download-volume", 0, "Syncs the Volume with an existing cloud volume")
	df := flagSet.String("download-file", "", "Downloads a file from the cloud to the local sdfs filesystem")
	dest := flagSet.String("destination", "", "The relative path for destination of the downloaded file")
	overwrite := flagSet.Bool("overwrite", false, "Overwrite the destination file if it exists")
	connection := ParseAndConnect(flagSet)
	defer connection.CloseConnection(ctx)

	if *scv {

		evt, err := connection.SyncCloudVolume(ctx, true)
		if err != nil {
			fmt.Printf("Unable to sync cloud volume: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Syncing with Cloud Completed %s \n", evt.ShortMsg)
		return

	}

	if IsFlagPassed("download-volume", flagSet) {
		evt, err := connection.SyncFromCloudVolume(ctx, *vcv, *overwrite)
		if err != nil {
			fmt.Printf("Unable to sync cloud volume from %d error: %v\n", *vcv, err)
			os.Exit(1)
		}
		fmt.Printf("Syncing with Cloud Completed %s \n", evt.ShortMsg)
		return
	}
	if IsFlagPassed("download-file", flagSet) {
		_dest := df
		if IsFlagPassed("destination", flagSet) {
			_dest = dest
		}
		evt, err := connection.GetCloudFile(ctx, *df, *_dest, *overwrite, true)
		if err != nil {
			fmt.Printf("Unable to sync cloud volume from %d error: %v\n", *vcv, err)
			os.Exit(1)
		}
		fmt.Printf("Syncing with Cloud Completed %s \n", evt.ShortMsg)
		return
	}

}
