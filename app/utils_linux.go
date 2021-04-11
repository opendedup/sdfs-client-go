package main

import (
	"os"
	"strconv"
	"syscall"
)

//GetPermissions returns permissions in a format SDFS can understand
func GetPermissionss(src string) (uid, gid int32, perm int, err error) {
	info, _ := os.Stat(src)

	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		uid = int32(stat.Uid)
		gid = int32(stat.Gid)
		b := strconv.FormatInt(int64(stat.Mode), 8)

		runeSample := []rune(b)
		l := len(runeSample)
		b = string(runeSample[l-3 : l])
		perm, err = strconv.Atoi(b)
		if err != nil {
			return -1, -1, -1, err
		}
	} else {
		// we are not in linux, this won't work anyway in windows,
		// but maybe you want to log warnings
		uid = int32(os.Getuid())
		gid = int32(os.Getgid())
		perm = 644
	}
	return uid, gid, perm, nil
}
