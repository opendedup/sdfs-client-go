package test

import (
	"context"

	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDedupeNewConnection(t *testing.T) {
	connection := connect(t, true)
	ctx, cancel := context.WithCancel(context.Background())
	defer connection.CloseConnection(ctx)
	defer cancel()
	assert.NotNil(t, connection)
}

func TestDedupeWriteFile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := connect(t, true)
	assert.NotNil(t, connection)
	defer connection.CloseConnection(ctx)
	fn, hash := makeFile(t, "", 1024, true)
	nhash := readFile(t, fn, false)
	assert.Equal(t, hash, nhash)

	err := connection.DeleteFile(ctx, fn)
	assert.Nil(t, err)
}

func TestDedupeWriteLargeFile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := connect(t, true)
	assert.NotNil(t, connection)
	defer connection.CloseConnection(ctx)
	fn, hash := makeLargeBlockFile(t, "", 768*1024, true, 1024)
	nhash := readFile(t, fn, false)
	assert.Equal(t, hash, nhash)

	err := connection.DeleteFile(ctx, fn)
	assert.Nil(t, err)
}

func TestDedupeReWriteFile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := connect(t, true)
	assert.NotNil(t, connection)

	fn, hash := makeFile(t, "", 1024*1024, true)
	nhash := readFile(t, fn, false)
	info, _ := connection.Stat(ctx, fn)
	dd := info.IoMonitor.ActualBytesWritten
	t.Logf("info %v", info)
	assert.Equal(t, hash, nhash)
	connection.CloseConnection(ctx)
	connection = connect(t, false)
	defer connection.CloseConnection(ctx)
	_, err := connection.Download(ctx, fn, "/tmp/"+fn)
	if err != nil {
		t.Logf("download error %v", err)
	}
	dn := string(randBytesMaskImpr(16))
	_, err = connection.Upload(ctx, "/tmp/"+fn, dn)
	if err != nil {
		t.Logf("upload error %v", err)
	}
	info, _ = connection.Stat(ctx, dn)
	t.Logf("info %v", info)
	assert.Greater(t, dd, info.IoMonitor.ActualBytesWritten)
	assert.Nil(t, err)
	dhash := readFile(t, fn, false)
	assert.Equal(t, hash, dhash)
	err = connection.DeleteFile(ctx, fn)
	assert.Nil(t, err)
	err = connection.DeleteFile(ctx, dn)
	assert.Nil(t, err)
}
