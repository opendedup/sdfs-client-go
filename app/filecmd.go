package main

import (
	context "context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"

	spb "github.com/opendedup/sdfs-client-go/sdfs"
)

//FileCmd Configure Volume functions for sdfscli
func FileCmd(ctx context.Context, flagSet *flag.FlagSet) {
	flagSet.Bool("upload", false, "Uploads a file to the filesystem")
	flagSet.Bool("download", false, "Downloads a file from the filesystem")
	flagSet.Bool("snapshot", false, "Creates a snapshot of a file")
	flagSet.Bool("rename", false, "Renames a file")
	flagSet.Bool("change-listener", false, "Listens and notifies on changes")
	src := flagSet.String("src", ".", "The source file")
	dst := flagSet.String("dst", ".", "The destination file")
	delfile := flagSet.String("delete", ".", "Deletes a file")
	mkdir := flagSet.String("mkdir", ".", "Creates a directory")
	linfo := flagSet.String("list", ".", "Returns File Info in list format")
	finfo := flagSet.String("detail", ".", "Returns Detailed File Info")
	fattr := flagSet.String("attributes", ".", "Returns File Attributes")
	sattr := flagSet.String("attribute", ".", "Sets A File Attribute")
	key := flagSet.String("key", "key", "The Attribute Key")
	value := flagSet.String("value", "value", "The Attribute Value")
	fio := flagSet.String("stats", ".", "Returns File Dedupe Rates and other IO Attributes")
	connection := ParseAndConnect(flagSet)
	if IsFlagPassed("mkdir", flagSet) {
		err := connection.MkDirAll(ctx, *mkdir)
		if err != nil {
			fmt.Printf("Unable to mkdir: %s error: %v\n", *mkdir, err)
			os.Exit(1)
		}
		fmt.Printf("Made directory %s \n", *mkdir)
		return
	}
	if IsFlagPassed("delete", flagSet) {
		err := connection.DeleteFile(ctx, *delfile)
		if err != nil {
			fmt.Printf("Unable to delete: %s error: %v\n", *delfile, err)
			os.Exit(1)
		}
		fmt.Printf("Deleted %s \n", *delfile)
		return
	}
	if IsFlagPassed("rename", flagSet) {
		if !IsFlagPassed("src", flagSet) || !IsFlagPassed("dst", flagSet) {
			fmt.Println("--src and dst must be set for rename")
			os.Exit(1)
		}
		err := connection.Rename(ctx, *src, *dst)
		if err != nil {
			fmt.Printf("Unable to rename from: %s to: %s error: %v\n", *src, *dst, err)
			os.Exit(1)
		}
		fmt.Printf("Renamed  %s to %s\n", *src, *dst)
		return
	}
	if IsFlagPassed("snapshot", flagSet) {
		if !IsFlagPassed("src", flagSet) || !IsFlagPassed("dst", flagSet) {
			fmt.Println("--src and dst must be set for snapshot")
			os.Exit(1)
		}
		_, err := connection.CopyFile(ctx, *src, *dst, false)
		if err != nil {
			fmt.Printf("Unable to create snapshot from: %s to: %s error: %v\n", *src, *dst, err)
			os.Exit(1)
		}
		fmt.Printf("Created snapshot of  %s to %s\n", *src, *dst)
		return
	}
	if IsFlagPassed("upload", flagSet) {
		if !IsFlagPassed("src", flagSet) {
			fmt.Println("--src must be set for upload")
			os.Exit(1)
		}
		if !IsFlagPassed("dst", flagSet) {
			dst = src
		}
		len, err := connection.Upload(ctx, *src, *dst)
		if err != nil {
			fmt.Printf("Unable to upload: %s to: %s error: %v\n", *src, *dst, err)
			os.Exit(1)
		}
		fmt.Printf("Uploaded %s, %d bytes written \n", *src, len)
		return
	}
	if IsFlagPassed("download", flagSet) {
		if !IsFlagPassed("src", flagSet) || !IsFlagPassed("dst", flagSet) {
			fmt.Println("--src must be set for download")
			os.Exit(1)
		}
		if !IsFlagPassed("dst", flagSet) {
			dst = src
		}
		len, err := connection.Download(ctx, *src, *dst)
		if err != nil {
			fmt.Printf("Unable to download: %s to: %s error: %v\n", *src, *dst, err)
			os.Exit(1)
		}
		fmt.Printf("Downloaded %s, %d bytes written \n", *src, len)
		return
	}
	if IsFlagPassed("attribute", flagSet) {
		if !IsFlagPassed("key", flagSet) && !IsFlagPassed("value", flagSet) {
			fmt.Println("--key and --value must be set")
			os.Exit(1)
		}

		err := connection.SetXAttr(ctx, *key, *value, *sattr)
		if err != nil {
			fmt.Printf("Unable to set extended attributes for: %v error: %v\n", *sattr, err)
			os.Exit(1)
		}
		fmt.Printf("Key and Value Set for: %s \n", *sattr)
		return
	}
	if IsFlagPassed("list", flagSet) {

		_, fInfo, err := connection.ListDir(ctx, *linfo, "", false, int32(1000000))
		if err != nil {
			fmt.Printf("Unable to get file info for: %v error: %v\n", *linfo, err)
			os.Exit(1)
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"File Name", "Logical Size", "Physical Size", "Dedupe Rate", "Modified", "Symlink", "Type"})
		for _, v := range fInfo {
			if v.Type == 0 {
				iom := v.IoMonitor
				dedupeRate := 100.0
				if iom.ActualBytesWritten > 0 && v.Size > 0 {
					dedupeRate = ((float64(v.Size) - float64(iom.ActualBytesWritten)) / float64(v.Size)) * 100
				}
				t := time.Unix(0, v.Mtime*int64(time.Millisecond))

				table.Append([]string{v.FileName,
					strconv.FormatInt(v.Size, 10),
					strconv.FormatInt(iom.ActualBytesWritten, 10),
					fmt.Sprintf("%.2f%%", dedupeRate),
					t.String(), fmt.Sprintf("%t", v.Symlink), "file"})

			} else {
				t := time.Unix(0, v.Mtime*int64(time.Millisecond))
				table.Append([]string{v.FileName,
					"",
					"",
					"",
					t.String(),
					fmt.Sprintf("%t", v.Symlink),
					"dir"})
			}
		}
		table.SetAlignment(tablewriter.ALIGN_LEFT)

		table.Render()
	}

	if IsFlagPassed("file-detail", flagSet) {

		_, fInfo, err := connection.ListDir(ctx, *finfo, "", false, int32(1000000))
		if err != nil {
			fmt.Printf("Unable to get file info for: %v error: %v\n", *finfo, err)
			os.Exit(1)
		}

		for _, v := range fInfo {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"File Name", v.FileName})
			atime := time.Unix(0, v.Atime*int64(time.Millisecond))
			ctime := time.Unix(0, v.Ctime*int64(time.Millisecond))
			mtime := time.Unix(0, v.Mtime*int64(time.Millisecond))
			table.Append([]string{"File Name", v.FileName})
			if v.Type == 0 {

				table.Append([]string{"Size", strconv.FormatInt(v.Size, 10)})
				table.Append([]string{"File GUID", v.FileGuild})
				table.Append([]string{"Map GUID", v.MapGuid})
				table.Append([]string{"File Path", v.FilePath})
				table.Append([]string{"Access Time", atime.String()})
				table.Append([]string{"Create Time", ctime.String()})
				table.Append([]string{"Modifies Time", mtime.String()})
				table.Append([]string{"Execute", fmt.Sprintf("%t", v.Execute)})
				table.Append([]string{"Read", fmt.Sprintf("%t", v.Read)})
				table.Append([]string{"Read", fmt.Sprintf("%t", v.Write)})
				table.Append([]string{"Hidden", fmt.Sprintf("%t", v.Hidden)})
				table.Append([]string{"Hash Code", fmt.Sprintf("%d", v.Hashcode)})
				table.Append([]string{"ID", v.Id})
				table.Append([]string{"Importing", fmt.Sprintf("%t", v.Importing)})
				table.Append([]string{"Unix Permissions", fmt.Sprintf("%d", v.Permissions)})
				table.Append([]string{"Group ID", fmt.Sprintf("%d", v.GroupId)})
				table.Append([]string{"User ID", fmt.Sprintf("%d", v.UserId)})
				table.Append([]string{"File Open", fmt.Sprintf("%t", v.Open)})
				table.Append([]string{"Symlink", fmt.Sprintf("%t", v.Symlink)})
				table.Append([]string{"Symlink Path", fmt.Sprintf("%s", v.SymlinkPath)})
				table.Append([]string{"File Type", fmt.Sprintf("%s", v.Type)})

			} else {
				table.Append([]string{"Size", strconv.FormatInt(v.Size, 10)})
				table.Append([]string{"File Path", v.FilePath})
				table.Append([]string{"Access Time", atime.String()})
				table.Append([]string{"Create Time", ctime.String()})
				table.Append([]string{"Modifies Time", mtime.String()})
				table.Append([]string{"Execute", fmt.Sprintf("%t", v.Execute)})
				table.Append([]string{"Read", fmt.Sprintf("%t", v.Read)})
				table.Append([]string{"Read", fmt.Sprintf("%t", v.Write)})
				table.Append([]string{"Hidden", fmt.Sprintf("%t", v.Hidden)})
				table.Append([]string{"Hash Code", fmt.Sprintf("%d", v.Hashcode)})
				table.Append([]string{"Unix Permissions", fmt.Sprintf("%d", v.Permissions)})
				table.Append([]string{"Group ID", fmt.Sprintf("%d", v.GroupId)})
				table.Append([]string{"User ID", fmt.Sprintf("%d", v.UserId)})
				table.Append([]string{"Symlink", fmt.Sprintf("%t", v.Symlink)})
				table.Append([]string{"Symlink Path", fmt.Sprintf("%s", v.SymlinkPath)})
				table.Append([]string{"File Type", fmt.Sprintf("%s", v.Type)})
			}
			table.SetAlignment(tablewriter.ALIGN_LEFT)

			table.Render()
		}

	}

	if IsFlagPassed("change-listener", flagSet) {
		c := make(chan *spb.FileMessageResponse)
		go connection.FileNotification(ctx, c)
		for {
			fInfo := <-c
			if fInfo == nil {
				fmt.Printf("done")
				return
			}
			for _, v := range fInfo.Response {
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"File Name", v.FileName})
				atime := time.Unix(0, v.Atime*int64(time.Millisecond))
				ctime := time.Unix(0, v.Ctime*int64(time.Millisecond))
				mtime := time.Unix(0, v.Mtime*int64(time.Millisecond))
				table.Append([]string{"File Name", v.FileName})
				if v.Type == 0 {

					table.Append([]string{"Size", strconv.FormatInt(v.Size, 10)})
					table.Append([]string{"File GUID", v.FileGuild})
					table.Append([]string{"Map GUID", v.MapGuid})
					table.Append([]string{"File Path", v.FilePath})
					table.Append([]string{"Access Time", atime.String()})
					table.Append([]string{"Create Time", ctime.String()})
					table.Append([]string{"Modifies Time", mtime.String()})
					table.Append([]string{"Execute", fmt.Sprintf("%t", v.Execute)})
					table.Append([]string{"Read", fmt.Sprintf("%t", v.Read)})
					table.Append([]string{"Read", fmt.Sprintf("%t", v.Write)})
					table.Append([]string{"Hidden", fmt.Sprintf("%t", v.Hidden)})
					table.Append([]string{"Hash Code", fmt.Sprintf("%d", v.Hashcode)})
					table.Append([]string{"ID", v.Id})
					table.Append([]string{"Importing", fmt.Sprintf("%t", v.Importing)})
					table.Append([]string{"Unix Permissions", fmt.Sprintf("%d", v.Permissions)})
					table.Append([]string{"Group ID", fmt.Sprintf("%d", v.GroupId)})
					table.Append([]string{"User ID", fmt.Sprintf("%d", v.UserId)})
					table.Append([]string{"File Open", fmt.Sprintf("%t", v.Open)})
					table.Append([]string{"Symlink", fmt.Sprintf("%t", v.Symlink)})
					table.Append([]string{"Symlink Path", fmt.Sprintf("%s", v.SymlinkPath)})
					table.Append([]string{"File Type", fmt.Sprintf("%s", v.Type)})

				} else {
					table.Append([]string{"Size", strconv.FormatInt(v.Size, 10)})
					table.Append([]string{"File Path", v.FilePath})
					table.Append([]string{"Access Time", atime.String()})
					table.Append([]string{"Create Time", ctime.String()})
					table.Append([]string{"Modifies Time", mtime.String()})
					table.Append([]string{"Execute", fmt.Sprintf("%t", v.Execute)})
					table.Append([]string{"Read", fmt.Sprintf("%t", v.Read)})
					table.Append([]string{"Read", fmt.Sprintf("%t", v.Write)})
					table.Append([]string{"Hidden", fmt.Sprintf("%t", v.Hidden)})
					table.Append([]string{"Hash Code", fmt.Sprintf("%d", v.Hashcode)})
					table.Append([]string{"Unix Permissions", fmt.Sprintf("%d", v.Permissions)})
					table.Append([]string{"Group ID", fmt.Sprintf("%d", v.GroupId)})
					table.Append([]string{"User ID", fmt.Sprintf("%d", v.UserId)})
					table.Append([]string{"Symlink", fmt.Sprintf("%t", v.Symlink)})
					table.Append([]string{"Symlink Path", fmt.Sprintf("%s", v.SymlinkPath)})
					table.Append([]string{"File Type", fmt.Sprintf("%s", v.Type)})
				}
				table.Append([]string{"Event Type", fmt.Sprintf("%s", fInfo.Action)})
				table.SetAlignment(tablewriter.ALIGN_LEFT)

				table.Render()
			}
		}
	}

	if IsFlagPassed("file-attributes", flagSet) {

		_, fInfo, err := connection.ListDir(ctx, *fattr, "", false, int32(1000000))
		if err != nil {
			fmt.Printf("Unable to get file info for: %v error: %v\n", *fattr, err)
			os.Exit(1)
		}

		for _, v := range fInfo {
			if v.Type == 0 {
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"File Name", v.FileName})
				table.Append([]string{"File Name", v.FileName})

				for _, attr := range v.FileAttributes {
					table.Append([]string{attr.Key, attr.Value})
				}
				table.SetAlignment(tablewriter.ALIGN_LEFT)

				table.Render()
			}

		}

	}

	if IsFlagPassed("file-io", flagSet) {

		_, fInfo, err := connection.ListDir(ctx, *fio, "", false, int32(1000000))
		if err != nil {
			fmt.Printf("Unable to get file info for: %v error: %v\n", *fio, err)
			os.Exit(1)
		}

		for _, v := range fInfo {
			if v.Type == 0 {
				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"File Name", v.FileName})
				table.Append([]string{"File Name", v.FileName})
				io := v.IoMonitor
				dedupeRate := 100.0
				if io.ActualBytesWritten > 0 && v.Size > 0 {
					dedupeRate = ((float64(v.Size) - float64(io.ActualBytesWritten)) / float64(v.Size)) * 100
				}

				table.Append([]string{"Dedup Rate", fmt.Sprintf("%.2f%%", dedupeRate)})
				table.Append([]string{"Physical Data Written", FormatSize(io.ActualBytesWritten)})
				table.Append([]string{"Read Data", FormatSize(io.BytesRead)})
				table.Append([]string{"Duplicate Data", FormatSize(io.DuplicateBlocks)})
				table.Append([]string{"Virtual Data Written", FormatSize(io.VirtualBytesWritten)})
				table.Append([]string{"Physical Bytes Written", fmt.Sprintf("%d", io.ActualBytesWritten)})
				table.Append([]string{"Bytes Read", fmt.Sprintf("%d", io.BytesRead)})
				table.Append([]string{"Duplicate Bytes", fmt.Sprintf("%d", io.DuplicateBlocks)})
				table.Append([]string{"Virtual Bytes Written", fmt.Sprintf("%d", io.VirtualBytesWritten)})
				table.Append([]string{"Read Ops", fmt.Sprintf("%d", io.ReadOpts)})
				table.Append([]string{"Write Ops", fmt.Sprintf("%d", io.WriteOpts)})

				table.SetAlignment(tablewriter.ALIGN_LEFT)

				table.Render()
			}
		}

	}

}
