package main

import (
	context "context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/blake2b"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func RandBytesMaskImpr(n int) []byte {
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

func TestMkDir(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := Tconnect(t)
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

func TestCreateFile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := Tconnect(t)
	assert.NotNil(t, connection)
	err := connection.MkNod(ctx, "testfile", 511, 0)
	assert.Nil(t, err)
	stat, err := connection.GetAttr(ctx, "testfile")
	assert.Nil(t, err)
	assert.Equal(t, stat.Mode, int32(511))
	err = connection.DeleteFile(ctx, "testfile")
	assert.Nil(t, err)
	stat, err = connection.GetAttr(ctx, "testfile")
	assert.NotNil(t, err)
	connection.CloseConnection(ctx)
}

func TestWriteFile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := Tconnect(t)
	assert.NotNil(t, connection)
	err := connection.MkNod(ctx, "testfile", 511, 0)
	assert.Nil(t, err)
	stat, err := connection.GetAttr(ctx, "testfile")
	assert.Nil(t, err)
	assert.Equal(t, stat.Mode, int32(511))
	fh, err := connection.Open(ctx, "testfile", 0)
	assert.Nil(t, err)
	b := []byte("I like cheese\n")
	err = connection.Write(ctx, fh, b, 0, int32(len(b)))
	assert.Nil(t, err)
	err = connection.Release(ctx, fh)
	assert.Nil(t, err)
	err = connection.DeleteFile(ctx, "testfile")
	assert.Nil(t, err)
	stat, err = connection.GetAttr(ctx, "testfile")
	assert.NotNil(t, err)
	connection.CloseConnection(ctx)
}
func TestWriteReadFile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := Tconnect(t)
	assert.NotNil(t, connection)
	err := connection.MkNod(ctx, "testfile", 511, 0)
	assert.Nil(t, err)

	stat, err := connection.GetAttr(ctx, "testfile")
	assert.Nil(t, err)

	assert.Equal(t, stat.Mode, int32(511))
	fh, err := connection.Open(ctx, "testfile", 0)
	assert.Nil(t, err)

	b := RandBytesMaskImpr(1024)
	err = connection.Write(ctx, fh, b, 0, int32(len(b)))
	assert.Nil(t, err)

	nb, err := connection.Read(ctx, fh, 0, int32(len(b)))
	assert.Nil(t, err)
	assert.Equal(t, nb, b)

	err = connection.Release(ctx, fh)
	assert.Nil(t, err)

	err = connection.DeleteFile(ctx, "testfile")
	assert.Nil(t, err)

	stat, err = connection.GetAttr(ctx, "testfile")
	assert.NotNil(t, err)

	connection.CloseConnection(ctx)
}

func TestLargeWriteFile(t *testing.T) {
	t.Run("test1", func(t *testing.T) {
		fileSize := int64(512 * 1024 * 1024)
		writeFile(t, 1, fileSize, true)
	})
}

func TestLargeWriteFiles(t *testing.T) {
	for i := 0; i < 10; i++ {
		fn := i
		t.Run("writetest"+fmt.Sprint(fn), func(t *testing.T) {
			t.Parallel()
			fileSize := int64(512 * 1024 * 1024)
			writeFile(t, fn, fileSize, true)
		})
	}
}

func TestChow(t *testing.T) {

}

func TestLargeWriteReadFile(t *testing.T) {
	fileSize := int64(512 * 1024 * 1024)
	wb := fmt.Sprintf("%x", writeFile(t, 1, fileSize, false))
	rb := fmt.Sprintf("%x", readFile(t, 1, true))
	assert.Equal(t, wb, rb, "Read and Write Hashes don't match ")
}

func TestLargeWriteReadFiles(t *testing.T) {
	for i := 0; i < 10; i++ {
		fn := i
		t.Run("readwritetest"+fmt.Sprint(fn), func(t *testing.T) {
			t.Parallel()
			fileSize := int64(1024 * 1024 * 1024)
			wb := fmt.Sprintf("%x", writeFile(t, fn, fileSize, false))
			rb := fmt.Sprintf("%x", readFile(t, fn, true))
			assert.Equal(t, wb, rb, "Read and Write Hashes don't match ")
		})
	}
}

func readFile(t *testing.T, i int, delete bool) []byte {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := Tconnect(t)
	assert.NotNil(t, connection)
	filenm := "testfile" + fmt.Sprint(i)
	stat, err := connection.GetAttr(ctx, filenm)
	assert.Nil(t, err)

	assert.Greater(t, stat.Size, int64(0))
	fh, err := connection.Open(ctx, filenm, 0)
	assert.Nil(t, err)
	maxoffset := stat.Size
	offset := int64(0)
	b := make([]byte, 0)
	h, err := blake2b.New(32, b)
	assert.Nil(t, err)
	readSize := int32(1024 * 1024)
	for offset < maxoffset {
		if readSize > int32(maxoffset-offset) {
			readSize = int32(maxoffset - offset)
		}
		b, err := connection.Read(ctx, fh, offset, int32(readSize))
		h.Write(b)
		assert.Nil(t, err)
		offset += int64(len(b))
		b = nil
	}
	err = connection.Release(ctx, fh)
	assert.Nil(t, err)

	if delete {
		err = connection.DeleteFile(ctx, filenm)
		assert.Nil(t, err)
		stat, err = connection.GetAttr(ctx, filenm)
		assert.NotNil(t, err)
	}

	connection.CloseConnection(ctx)
	bs := h.Sum(nil)
	return bs
}

func writeFile(t *testing.T, i int, size int64, delete bool) []byte {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	connection := Tconnect(t)
	assert.NotNil(t, connection)
	filenm := "testfile" + fmt.Sprint(i)
	err := connection.MkNod(ctx, filenm, 511, 0)
	assert.Nil(t, err)

	stat, err := connection.GetAttr(ctx, filenm)
	assert.Nil(t, err)

	assert.Equal(t, stat.Mode, int32(511))
	fh, err := connection.Open(ctx, filenm, 0)
	assert.Nil(t, err)
	maxoffset := size
	offset := int64(0)
	h, err := blake2b.New(32, make([]byte, 0))
	assert.Nil(t, err)
	for offset < maxoffset {
		b := RandBytesMaskImpr(1024 * 32)
		err = connection.Write(ctx, fh, b, offset, int32(len(b)))
		h.Write(b)
		assert.Nil(t, err)
		offset += int64(len(b))
		b = nil
	}

	stat, err = connection.GetAttr(ctx, filenm)
	assert.Equal(t, stat.Size, maxoffset)
	err = connection.Release(ctx, fh)
	assert.Nil(t, err)
	if delete {
		err = connection.DeleteFile(ctx, filenm)
		assert.Nil(t, err)
		stat, err = connection.GetAttr(ctx, filenm)
		assert.NotNil(t, err)
	}

	connection.CloseConnection(ctx)
	bs := h.Sum(nil)
	return bs
}
