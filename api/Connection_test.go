package api

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	network "github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	natting "github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/blake2b"
)

const (
	address     = "sdfss://localhost:6442"
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func randBytesMaskImpr(n int) []byte {
	rand.Seed(time.Now().UTC().UnixNano())
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return b
}

func TestNewConnection(t *testing.T) {
	connection := connect(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer connection.CloseConnection(ctx)
	defer cancel()
	assert.NotNil(t, connection)
}

func TestChow(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := connect(t)
	defer connection.CloseConnection(ctx)
	assert.NotNil(t, connection)
	fn, _ := makeFile(t, "", 128)
	err := connection.Chown(ctx, fn, int32(100), int32(100))
	assert.Nil(t, err)
	stat, err := connection.GetAttr(ctx, fn)
	assert.Nil(t, err)
	assert.Equal(t, stat.Gid, int32(100))
	assert.Equal(t, stat.Uid, int32(100))
	deleteFile(t, fn)
}

func TestMkNod(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := connect(t)
	defer connection.CloseConnection(ctx)
	assert.NotNil(t, connection)
	fn, _ := makeFile(t, "", 128)
	deleteFile(t, fn)
}

func TestMkDir(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := connect(t)
	defer connection.CloseConnection(ctx)
	assert.NotNil(t, connection)
	err := connection.MkDir(ctx, "testdir", 511)
	assert.Nil(t, err)
	stat, err := connection.GetAttr(ctx, "testdir")
	assert.Nil(t, err)
	assert.Equal(t, stat.Mode, int32(16895))
	err = connection.RmDir(ctx, "testdir")
	assert.Nil(t, err)
	stat, err = connection.GetAttr(ctx, "testdir")
	assert.NotNil(t, err)
	connection.CloseConnection(ctx)
}

func TestMkDirAll(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := connect(t)
	defer connection.CloseConnection(ctx)
	assert.NotNil(t, connection)
	err := connection.MkDirAll(ctx, "testdir/t")
	assert.Nil(t, err)
	stat, err := connection.GetAttr(ctx, "testdir/t")
	assert.Nil(t, err)
	assert.Equal(t, stat.Mode, int32(16832))
	err = connection.RmDir(ctx, "testdir/t")
	assert.Nil(t, err)
	stat, err = connection.GetAttr(ctx, "testdir/t")
	assert.NotNil(t, err)
	err = connection.RmDir(ctx, "testdir")
	assert.Nil(t, err)
	stat, err = connection.GetAttr(ctx, "testdir")
	assert.NotNil(t, err)
	connection.CloseConnection(ctx)
}

func TestListDir(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := connect(t)
	assert.NotNil(t, connection)
	defer connection.CloseConnection(ctx)
	dn := string(randBytesMaskImpr(16))
	err := connection.MkDir(ctx, dn, 511)
	assert.Nil(t, err)
	var files []string
	for i := 0; i < 10; i++ {
		fn, _ := makeFile(t, dn, 1024)
		files = append(files, fn)
	}
	_, list, err := connection.ListDir(ctx, dn, "", false, 20)
	assert.Nil(t, err)
	var afiles []string
	for _, l := range list {
		afiles = append(afiles, l.FilePath)
		connection.DeleteFile(ctx, l.FilePath)
	}
	assert.ElementsMatch(t, files, afiles)
	err = connection.RmDir(ctx, dn)
	assert.Nil(t, err)
	_, err = connection.GetAttr(ctx, dn)
	assert.NotNil(t, err)
}

func TestCleanStore(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := connect(t)
	assert.NotNil(t, connection)
	defer connection.CloseConnection(ctx)
	var files []string
	for i := 0; i < 10; i++ {
		fn, _ := makeFile(t, "", 1024*1024)
		files = append(files, fn)
	}
	info, err := connection.DSEInfo(ctx)
	assert.Nil(t, err)
	sz := info.CurrentSize
	for _, l := range files {
		connection.DeleteFile(ctx, l)
	}
	time.Sleep(10 * time.Second)
	connection.CleanStore(ctx, true, true)
	info, err = connection.DSEInfo(ctx)
	nsz := info.CurrentSize
	assert.NotEqual(t, sz, nsz)
	t.Logf("orig = %d new = %d", sz, nsz)
}

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UTC().UnixNano())
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Printf("Unable to create docker client %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cli.NegotiateAPIVersion(ctx)
	imagename := "gcr.io/hybrics/hybrics:3.11"
	containername := string(randBytesMaskImpr(16))
	portopening := "6442"
	inputEnv := []string{fmt.Sprintf("CAPACITY=%s", "1TB"), fmt.Sprintf("EXTENDED_CMD=--hashtable-rm-threshold=1000")}
	_, err = runContainer(cli, imagename, containername, portopening, inputEnv)
	if err != nil {
		fmt.Printf("Unable to create docker client %v", err)
	}
	DisableTrust = true
	connection, err := NewConnection(address)
	retrys := 0
	for err != nil {
		log.Printf("retries = %d", retrys)
		time.Sleep(20 * time.Second)
		connection, err = NewConnection(address)
		if retrys > 10 {
			break
		} else {
			retrys++
		}
	}
	if connection != nil {
		connection.CloseConnection(ctx)
	}

	if err != nil {
		fmt.Printf("Unable to create connection %v", err)
	}
	code := m.Run()
	//stopAndRemoveContainer(cli, containername)
	os.Exit(code)
}

func runContainer(client *client.Client, imagename string, containername string, port string, inputEnv []string) (string, error) {
	// Define a PORT opening
	newport, err := natting.NewPort("tcp", port)
	if err != nil {
		fmt.Println("Unable to create docker port")
		return "", err
	}

	// Configured hostConfig:
	// https://godoc.org/github.com/docker/docker/api/types/container#HostConfig
	hostConfig := &container.HostConfig{
		PortBindings: natting.PortMap{
			newport: []natting.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: port,
				},
			},
		},
		RestartPolicy: container.RestartPolicy{
			Name: "always",
		},
		LogConfig: container.LogConfig{
			Type:   "json-file",
			Config: map[string]string{},
		},
	}

	// Define Network config (why isn't PORT in here...?:
	// https://godoc.org/github.com/docker/docker/api/types/network#NetworkingConfig
	networkConfig := &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{},
	}
	gatewayConfig := &network.EndpointSettings{
		Gateway: "gatewayname",
	}
	networkConfig.EndpointsConfig["bridge"] = gatewayConfig

	// Define ports to be exposed (has to be same as hostconfig.portbindings.newport)
	exposedPorts := map[natting.Port]struct{}{
		newport: struct{}{},
	}

	// Configuration
	// https://godoc.org/github.com/docker/docker/api/types/container#Config
	config := &container.Config{
		Image:        imagename,
		Env:          inputEnv,
		ExposedPorts: exposedPorts,
		Hostname:     fmt.Sprintf("%s-hostnameexample", imagename),
	}

	// Creating the actual container. This is "nil,nil,nil" in every example.
	cont, err := client.ContainerCreate(
		context.Background(),
		config,
		hostConfig,
		networkConfig, nil,
		containername,
	)

	if err != nil {
		log.Println(err)
		return "", err
	}

	// Run the actual container
	client.ContainerStart(context.Background(), cont.ID, types.ContainerStartOptions{})
	log.Printf("Container %s is created", cont.ID)

	return cont.ID, nil
}

func stopAndRemoveContainer(client *client.Client, containername string) error {
	ctx := context.Background()

	if err := client.ContainerStop(ctx, containername, nil); err != nil {
		log.Printf("Unable to stop container %s: %s", containername, err)
	}

	removeOptions := types.ContainerRemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}

	if err := client.ContainerRemove(ctx, containername, removeOptions); err != nil {
		log.Printf("Unable to remove container: %s", err)
		return err
	}

	return nil
}

func makeFile(t *testing.T, parent string, size int64) (string, []byte) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := connect(t)
	defer connection.CloseConnection(ctx)
	assert.NotNil(t, connection)
	fn := fmt.Sprintf("%s/%s", parent, string(randBytesMaskImpr(16)))
	err := connection.MkNod(ctx, fn, 511, 0)
	assert.Nil(t, err)
	stat, err := connection.GetAttr(ctx, fn)
	assert.Nil(t, err)
	assert.Equal(t, stat.Mode, int32(511))
	fh, err := connection.Open(ctx, fn, 0)
	assert.Nil(t, err)
	maxoffset := size
	offset := int64(0)
	h, err := blake2b.New(32, make([]byte, 0))
	assert.Nil(t, err)
	blockSz := 1024 * 32
	for offset < maxoffset {
		if blockSz > int(maxoffset-offset) {
			blockSz = int(maxoffset - offset)
		}
		b := randBytesMaskImpr(blockSz)
		err = connection.Write(ctx, fh, b, offset, int32(len(b)))
		h.Write(b)
		assert.Nil(t, err)
		offset += int64(len(b))
		b = nil
	}
	stat, err = connection.GetAttr(ctx, fn)
	assert.Equal(t, stat.Size, maxoffset)
	err = connection.Release(ctx, fh)
	return fn, h.Sum(nil)
}

func deleteFile(t *testing.T, fn string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := connect(t)
	defer connection.CloseConnection(ctx)
	assert.NotNil(t, connection)
	err := connection.DeleteFile(ctx, fn)
	assert.Nil(t, err)
	_, err = connection.GetAttr(ctx, fn)
	assert.NotNil(t, err)
}

func connect(t *testing.T) *SdfsConnection {
	DisableTrust = true
	connection, err := NewConnection(address)

	if err != nil {
		t.Errorf("Unable to connect to %s error: %v\n", address, err)
		return nil
	}
	return connection
}
