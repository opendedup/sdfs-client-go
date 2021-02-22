package main

import (
	context "context"
	"testing"

	pb "github.com/opendedup/sdfs-client-go/api"
	"github.com/stretchr/testify/assert"
)

var address string = "sdfss://localhost:6442"

func TestConnection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := Tconnect(t)
	assert.NotNil(t, connection)
	connection.CloseConnection(ctx)
}

func TestVolumeInfo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := Tconnect(t)
	assert.NotNil(t, connection)
	info, err := connection.GetVolumeInfo(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, info)
	connection.CloseConnection(ctx)
}

func Tconnect(t *testing.T) *pb.SdfsConnection {
	pb.DisableTrust = true
	connection, err := pb.NewConnection(address)

	if err != nil {
		t.Errorf("Unable to connect to %s error: %v\n", address, err)
		return nil
	}
	return connection
}
