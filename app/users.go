package main

import (
	context "context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/opendedup/sdfs-client-go/utils"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var myFlags arrayFlags

//CloudCmd cloud configuration functions for sdfscli
func UserCmd(ctx context.Context, flagSet *flag.FlagSet) {
	password := flagSet.String("user-password", "", "A password for the user that is the target")
	description := flagSet.String("description", "", "A description for the user")
	flagSet.Var(&myFlags, "permission", "A permssion to set for this user. This can be called multiple times."+
		" Possible permssions are  METADATA_READ,METADATA_WRITE,FILE_READ,FILE_WRITE,FILE_DELETE,VOLUME_READ,CONFIG_READ,"+
		"CONFIG_WRITE,EVENT_READ,AUTH_READ,AUTH_WRITE,ADMIN")
	add := flagSet.String("add", "", "add a user")
	del := flagSet.String("delete", "", "delete a user")
	perms := flagSet.String("set-permissions", "", "set permissions for a user")
	list := flagSet.Bool("list", false, "list users")
	spwd := flagSet.String("set-password", "", "set password for a user")
	connection := utils.ParseAndConnect(flagSet)
	defer connection.CloseConnection(ctx)

	if *list {
		lst, err := connection.ListUsers(ctx)
		if err != nil {
			fmt.Printf("Unable to list users: %v\n", err)
			os.Exit(1)
		}
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"User", "Description", "Permissions", "Last Login", "Last Failed Login"})

		for _, v := range lst {
			lt := time.Unix(0, v.LastLogin*int64(time.Millisecond))
			et := time.Unix(0, v.LastFailedLogin*int64(time.Millisecond))
			table.Append([]string{v.User,
				v.Description,
				fmt.Sprintf("%v", v.Permissions), lt.String(), et.String()})
		}
		table.SetAlignment(tablewriter.ALIGN_LEFT)

		table.Render()
		return

	} else if utils.IsFlagPassed("add", flagSet) {
		err := connection.AddUser(ctx, *add, *password, *description, myFlags)
		if err != nil {
			fmt.Printf("Unable to add user %s error: %v\n", *add, err)
			os.Exit(1)
		}
		fmt.Printf("Added user %s \n", *add)
		return
	} else if utils.IsFlagPassed("delete", flagSet) {
		err := connection.DeleteUser(ctx, *del)
		if err != nil {
			fmt.Printf("Unable to delete user %s error: %v\n", *del, err)
			os.Exit(1)
		}
		fmt.Printf("Deleted user %s \n", *del)
		return
	} else if utils.IsFlagPassed("spwd", flagSet) {
		err := connection.SetSdfsPassword(ctx, *spwd, *password)
		if err != nil {
			fmt.Printf("Unable to delete user %s error: %v\n", *spwd, err)
			os.Exit(1)
		}
		fmt.Printf("Deleted user %s \n", *spwd)
		return
	} else if utils.IsFlagPassed("set-permissions", flagSet) {
		err := connection.SetSdfsPermissions(ctx, *perms, myFlags)
		if err != nil {
			fmt.Printf("Unable to set permsissions for user %s error: %v\n", *perms, err)
			os.Exit(1)
		}
		fmt.Printf("Set permissions for user %s \n", *perms)
		return
	}

}
