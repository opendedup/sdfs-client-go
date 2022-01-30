package utils

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/user"
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
		return 0, fmt.Errorf("unable to parse string. Size must be set as \"<unit> <unit type>\" e.g \"10 TB\"")
	} else {
		sz, err := strconv.ParseInt(tokens[0], 10, 64)
		if err != nil {
			return 0, err
		}
		return unitmap[tokens[1]] * sz, nil
	}
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
	pwd := flagSet.String("p", "Password", "The Password for the Volume")
	u := flagSet.String("u", "admin", "The user to authenticate for this operation")
	address := flagSet.String("address", "sdfss://localhost:6442", "The address for the Volume")
	disableTrust := flagSet.Bool("trust-all", false, "Trust Self Signed TLS Certs")
	trustCert := flagSet.Bool("trust-cert", false, "Trust the certificate for url specified. This will download and store the certificate in $HOME/.sdfs/keys")
	mtls := flagSet.Bool("mtls", false, "Use Mutual TLS. This will use the certs located in $HOME/.sdfs/keys/[client.crt,client.key,ca.crt]"+
		"unless otherwise specified")
	mtlsca := flagSet.String("root-ca", "", "The path the CA cert used to sign the MTLS Cert. This defaults to $HOME/.sdfs/keys/ca.crt")
	mtlskey := flagSet.String("mtls-key", "", "The path the private used for mutual TLS. This defaults to $HOME/.sdfs/keys/client.key")
	mtlscert := flagSet.String("mtls-cert", "", "The path the client cert used for mutual TLS. This defaults to $HOME/.sdfs/keys/client.crt")
	dedupe := flagSet.Bool("dedupe", false, "Enable Client Side Dedupe")
	volumeid := flagSet.Int64("volumeID", -1, "The volume id to connect to. Required for access through proxy")
	compress := flagSet.Bool("compress", true, "Compress api traffic")
	flagSet.Parse(os.Args[2:])

	if !IsFlagPassed("address", flagSet) {
		address, err := getAddress()
		if err != nil {
			fmt.Printf("Error getting address for %s error: %v\n", *address, err)
			os.Exit(1)
		}
	}
	if *trustCert {
		err := pb.AddTrustedCert(*address)
		if err != nil {
			fmt.Printf("Error trusting certificate for %s error: %v\n", *address, err)
			os.Exit(1)
		}
	}
	if IsFlagPassed("root-ca", flagSet) {
		pb.MtlsCACert = *mtlsca
	}
	if IsFlagPassed("mtls-key", flagSet) {
		pb.MtlsKey = *mtlskey
	}
	if IsFlagPassed("mtls-cert", flagSet) {
		pb.MtlsCert = *mtlscert
	}

	if IsFlagPassed("p", flagSet) {
		pb.UserName = *u
		pb.Password = *pwd

	}
	if *disableTrust {
		//fmt.Println("TLS Verification Disabled")
		pb.DisableTrust = *disableTrust
	}
	if *mtls {
		//fmt.Println("Using Mutual TLS")
		pb.Mtls = *mtls
	}
	//fmt.Printf("Connecting to %s\n", *address)
	connection, err := pb.NewConnection(*address, *dedupe, *compress, *volumeid)
	if err != nil {
		fmt.Printf("Unable to connect to %s error: %v\n", *address, err)
		os.Exit(1)
	}

	return connection
}

//SdfsURL parses the credentials json and returns the url
type SdfsURL struct {
	URL string `json:"url"`
}

func getAddress() (url *string, err error) {

	user, err := user.Current()
	if err != nil {
		return url, err
	}
	filepath := user.HomeDir + "/.sdfs/credentials.json"
	purl, _ := os.LookupEnv("SDFS_URL")
	epath, eok := os.LookupEnv("SDFS_CREDENTIALS_PATH")
	if len(purl) > 0 {
		return &purl, nil
	} else if eok {
		filepath = epath
	}
	_, err = os.Stat(filepath)
	if os.IsNotExist(err) {
		purl = "sdfss://localhost:6442"
		return &purl, nil
	}
	jsonFile, err := os.Open(filepath)
	if err != nil {
		return url, err
	}
	// we initialize our Users array
	var jurl SdfsURL
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return url, err
	}
	err = json.Unmarshal(byteValue, &jurl)
	if err != nil {
		fmt.Printf("unable to parse %s", filepath)
		return url, err
	}
	if (len(jurl.URL)) > 0 {
		url = &jurl.URL
	} else {
		purl := "sdfss://localhost:6442"
		url = &purl
	}

	return url, nil
}

func GetPermissions(src string) (uid, gid int32, perm int, err error) {
	uid = int32(0)
	gid = int32(0)
	perm = 644
	return uid, gid, perm, nil
}
