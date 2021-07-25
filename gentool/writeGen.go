package main

import (
	"bytes"
	context "context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"encoding/json"

	"cloud.google.com/go/pubsub"
	"github.com/opendedup/sdfs-client-go/api"
	"github.com/opendedup/sdfs-client-go/utils"
)

type Cmd struct {
	StartTime int64  `json:"start"`
	Cmd       string `json:"cmd"`
	EndTime   int64  `json:"end"`
	Output    string `json:"output"`
}

type FileWrite struct {
	StartTime       int64  `json:"start"`
	ConcurrentFiles int    `json:"files"`
	Directory       string `json:"dir"`
	Size            int64  `json:"totalsize"`
	EndTime         int64  `json:"end"`
}

type Run struct {
	StartTime int64      `json:"start"`
	HostName  string     `json:"hostname"`
	EndTime   int64      `json:"end"`
	PreCmd    *Cmd       `json:"precmd"`
	FileWrite *FileWrite `json:"filewrite"`
	PostCmd   *Cmd       `json:"postcmd"`
	Local     bool       `json:"local"`
	OS        string     `json:"os"`
}

type Runs struct {
	Runs []Run `json:"runs"`
}

//FileCmd Configure Volume functions for sdfscli
func WriteCmd(ctx context.Context, flagSet *flag.FlagSet) {
	tempdir, err := ioutil.TempDir("", "sdfs")
	if err != nil {
		log.Fatal(err)
	}
	hdirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	flagSet.Bool("local", false, "Write to a local directory")
	directory := flagSet.String("write-dir", tempdir, "The local directory to write to.")
	recordFile := flagSet.String("record-file", fmt.Sprintf("%s/gentool.json", hdirname), "The file to capture all the run data in json format.")
	size := flagSet.String("size", "1 GB", "The size of each file to write")
	prewrite := flagSet.String("pre-run-cmd", "", "A command to run before a single run")
	postwrite := flagSet.String("post-run-cmd", "", "A command to run after a single run")
	files := flagSet.Int("concurrent-files", 1, "Then number of concurrent files to write to.")
	runs := flagSet.Int("runs", 1, "Then number of runs to do.")
	pubsubt := flagSet.String("pubsub-topic", "", "Publish IO Results to a pubsub topic")
	flagSet.Parse(os.Args[2:])
	var topic *pubsub.Topic
	if utils.IsFlagPassed("pubsub-topic", flagSet) {
		elm := strings.Split(*pubsubt, "/")
		client, err := pubsub.NewClient(ctx, elm[1])
		if err != nil {
			log.Errorf("error while creating pubsub client for : %s", elm[1])
			log.Fatal(err)
		}
		topic = client.Topic(elm[3])
	}
	for run := 0; run < *runs; run++ {
		hname, err := os.Hostname()
		if err != nil {
			log.Fatalf("Unable to get hostname %v\n", err)
			os.Exit(1)
		}
		runrec := &Run{StartTime: time.Now().Unix(), HostName: hname, OS: runtime.GOOS}
		szBytes, err := utils.GetSize(*size)
		log.Infof("will write: %d, size: %d(b), runs: %d", *files, szBytes, *runs)
		if err != nil {
			log.Fatalf("Unable to parse size: %s, %v\n", *size, err)
			os.Exit(1)
		}
		if utils.IsFlagPassed("pre-run-cmd", flagSet) {
			sts := time.Now().Unix()
			cmd := exec.Command(*prewrite)
			var out bytes.Buffer
			cmd.Stdout = &out
			err := cmd.Run()
			if err != nil {
				log.Errorf("error while executing : %s", *prewrite)
				log.Fatal(err)
			}
			log.Infof("pre-write command output: %s\n", out.String())
			pcmd := &Cmd{StartTime: sts, Output: out.String(), EndTime: time.Now().Unix(), Cmd: *prewrite}
			runrec.PreCmd = pcmd
		}
		if utils.IsFlagPassed("local", flagSet) {
			sts := time.Now().Unix()
			log.Infof("local writer\n")
			if _, err := os.Stat(*directory); os.IsNotExist(err) {
				os.MkdirAll(*directory, 0511)
			}
			var wg sync.WaitGroup
			for i := 0; i < *files; i++ {
				wg.Add(1)
				go writeToFolder(&wg, *directory, szBytes)
			}
			wg.Wait()
			fwrt := &FileWrite{StartTime: sts, ConcurrentFiles: *files, Directory: *directory, Size: int64(*files) * szBytes, EndTime: time.Now().Unix()}
			runrec.FileWrite = fwrt
		} else {
			log.Infof("sdfs writer\n")
			var wg sync.WaitGroup
			for i := 0; i < *files; i++ {
				wg.Add(1)
				connection := utils.ParseAndConnect(flagSet)
				defer connection.CloseConnection(ctx)
				go writeToSdfs(ctx, &wg, *directory, connection, szBytes)
			}
			wg.Wait()

		}
		if utils.IsFlagPassed("post-run-cmd", flagSet) {
			sts := time.Now().Unix()
			cmd := exec.Command(*postwrite)
			var out bytes.Buffer
			cmd.Stdout = &out
			err := cmd.Run()
			if err != nil {
				log.Errorf("error while executing : %s", *postwrite)
				log.Fatal(err)
			}
			log.Infof("post-write command output: %s\n", out.String())
			postrn := &Cmd{StartTime: sts, Output: out.String(), EndTime: time.Now().Unix(), Cmd: *postwrite}
			runrec.PostCmd = postrn

		}
		runrec.EndTime = time.Now().Unix()
		if utils.IsFlagPassed("pubsub-topic", flagSet) {

			data, err := json.MarshalIndent(runrec, "", " ")
			if err != nil {
				fmt.Println(err)
			}
			topic.Publish(ctx, &pubsub.Message{
				Data: data})

		} else {
			jsonFile, err := os.Open(*recordFile)
			var runs Runs
			if err != nil {
				runs = Runs{}
			} else {
				byteValue, _ := ioutil.ReadAll(jsonFile)

				json.Unmarshal(byteValue, &runs)
			}

			runs.Runs = append(runs.Runs, *runrec)
			jsonFile.Close()
			data, err := json.MarshalIndent(runs, "", " ")
			if err != nil {
				fmt.Println(err)
			}
			err = ioutil.WriteFile(*recordFile, data, 0644)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func writeToSdfs(ctx context.Context, wg *sync.WaitGroup, temdir string, connection *api.SdfsConnection, size int64) {
	defer wg.Done()
	fn, _, err := utils.WriteSdfsFile(ctx, connection, temdir, size, 4)
	if err != nil {
		log.Errorf("error while writing to %s", temdir)
		log.Fatal(err)
	}
	log.Infof("wrote to %s", fn)

}

func writeToFolder(wg *sync.WaitGroup, temdir string, size int64) {
	defer wg.Done()
	fn, _, err := utils.WriteLocalFile(temdir, size, 4)
	if err != nil {
		log.Errorf("error while writing to %s", temdir)
		log.Fatal(err)
	}
	log.Infof("wrote to %s", *fn)

}
