package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	pb "github.com/opendedup/sdfs-client-go/api"
)

var (
	suffixes = [6]string{"B", "KB", "MB", "GB", "TB", "PB"}
	unitmap  = make(map[string]int64)
)

func round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}

//GetSize Parses the size in bytes based on a string
func GetSize(size string) (int64, error) {
	unitmap["B"] = 1
	unitmap["KB"] = 1024
	unitmap["MB"] = 1024 * 1024
	unitmap["GB"] = 1024 * 1024 * 1024
	unitmap["TB"] = 1024 * 1024 * 1024 * 1024
	unitmap["PB"] = 1024 * 1024 * 1024 * 1024 * 1024
	size = strings.ToUpper(size)
	tokens := strings.Split(size, " ")
	if len(tokens) != 2 {
		return 0, fmt.Errorf("Unable to Parse String. Size must be set as \"<unit> <unit type>\" e.g \"10 TB\"")
	} else {
		sz, err := strconv.ParseInt(tokens[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unitmap[tokens[1]] * sz, nil
	}
}

//FormatSize Formats Size to String
func FormatSize(size int64) string {
	if size <= 0 {
		return "0 B"
	}
	suffixes[0] = "B"
	suffixes[1] = "KB"
	suffixes[2] = "MB"
	suffixes[3] = "GB"
	suffixes[4] = "TB"

	base := math.Log(float64(size)) / math.Log(1024)
	getSize := round(math.Pow(1024, base-math.Floor(base)), .5, 2)
	getSuffix := suffixes[int(math.Floor(base))]
	return strconv.FormatFloat(getSize, 'f', -1, 64) + " " + string(getSuffix)
}

//IsFlagPassed Check if the flags passed to flagset
func IsFlagPassed(name string, flagset *flag.FlagSet) bool {
	found := false
	flagset.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

//ParseAndConnect Parse Arguents and Connect to Volume
func ParseAndConnect(flagSet *flag.FlagSet) *pb.SdfsConnection {
	pwd := flagSet.String("pwd", "Password", "The Password for the Volume")
	address := flagSet.String("address", "sdfss://localhost:50051", "The Password for the Volume")
	disableTrust := flagSet.Bool("trust-all", false, "Trust Self Signed TLS Certs")
	flagSet.Parse(os.Args[2:])
	if IsFlagPassed("pwd", flagSet) {
		pb.UserName = "admin"
		pb.Password = *pwd

	}
	if *disableTrust {
		pb.DisableTrust = *disableTrust
	}
	connection, err := pb.NewConnection(*address)
	if err != nil {
		fmt.Printf("Unable to connect to %v error: %v\n", address, err)
		os.Exit(1)
	}

	return connection
}
