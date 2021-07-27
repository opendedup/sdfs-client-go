package utils

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/opendedup/sdfs-client-go/api"
	"github.com/seehuhn/mt19937"
	"golang.org/x/crypto/blake2b"
)

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	tb            = int64(1099511627776)
	gb            = int64(1073741824)
)

func RandBytesMaskImpr(n int) []byte {
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

func randBytes(n int, r *rand.Rand) []byte {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, r.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = r.Int63(), letterIdxMax
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

func WriteLocalFile(parent string, size int64, blocksize int) (*string, *[]byte, error) {
	rng := rand.New(mt19937.New())
	rng.Seed(time.Now().UnixNano())
	fn := fmt.Sprintf("%s/%s", parent, string(randBytes(16, rng)))
	f, err := os.Create(fn)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	maxoffset := size
	offset := int64(0)
	h, err := blake2b.New(32, make([]byte, 0))
	if err != nil {
		return nil, nil, err
	}
	blockSz := 1024 * blocksize
	for offset < maxoffset {
		if blockSz > int(maxoffset-offset) {
			blockSz = int(maxoffset - offset)
		}
		b := randBytes(blockSz, rng)
		_, err = f.WriteAt(b, offset)
		if err != nil {
			return nil, nil, err
		}
		h.Write(b)
		offset += int64(len(b))
		b = nil
	}
	hash := h.Sum(nil)
	_, err = f.WriteAt(hash, offset)
	if err != nil {
		return nil, nil, err
	}

	return &fn, &hash, nil
}

func WriteSdfsFile(ctx context.Context, connection *api.SdfsConnection, parent string, size int64, blocksize int) (*string, *[]byte, error) {
	fn := fmt.Sprintf("%s/%s", parent, string(RandBytesMaskImpr(16)))
	err := connection.MkNod(ctx, fn, 511, 0)
	if err != nil {
		return nil, nil, err
	}
	_, err = connection.GetAttr(ctx, fn)
	if err != nil {
		return nil, nil, err
	}
	fh, err := connection.Open(ctx, fn, 0)
	if err != nil {
		return nil, nil, err
	}
	maxoffset := size
	offset := int64(0)
	h, err := blake2b.New(32, make([]byte, 0))
	if err != nil {
		return nil, nil, err
	}
	blockSz := 1024 * blocksize
	for offset < maxoffset {
		if blockSz > int(maxoffset-offset) {
			blockSz = int(maxoffset - offset)
		}
		b := RandBytesMaskImpr(blockSz)
		err = connection.Write(ctx, fh, b, offset, int32(len(b)))
		if err != nil {
			return nil, nil, err
		}
		h.Write(b)
		offset += int64(len(b))
		b = nil
	}
	hash := h.Sum(nil)
	err = connection.Write(ctx, fh, hash, offset, int32(len(hash)))
	if err != nil {
		return nil, nil, err
	}
	_ = connection.Release(ctx, fh)

	return &fn, &hash, nil
}

func ReadSdfsFile(connection *api.SdfsConnection, filenm string, delete bool) ([]byte, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stat, err := connection.GetAttr(ctx, filenm)
	if err != nil {
		return nil, err
	}
	fh, err := connection.Open(ctx, filenm, 0)
	if err != nil {
		return nil, err
	}
	maxoffset := stat.Size - 32
	offset := int64(0)
	b := make([]byte, 0)
	h, err := blake2b.New(32, b)
	if err != nil {
		return nil, err
	}
	readSize := int32(1024 * 1024)
	for offset < maxoffset {
		if readSize > int32(maxoffset-offset) {
			readSize = int32(maxoffset - offset)
		}
		b, err := connection.Read(ctx, fh, offset, int32(readSize))
		h.Write(b)
		if err != nil {
			return nil, err
		}
		offset += int64(len(b))
		b = nil
	}
	err = connection.Release(ctx, fh)
	if err != nil {
		return nil, err
	}
	if delete {
		err = connection.DeleteFile(ctx, filenm)
		if err != nil {
			return nil, err
		}
		_, err = connection.GetAttr(ctx, filenm)
		if err != nil {
			return nil, err
		}
	}

	connection.CloseConnection(ctx)
	bs := h.Sum(nil)
	return bs, nil
}

func DeleteSdfsFile(connection *api.SdfsConnection, fn string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer connection.CloseConnection(ctx)
	err := connection.DeleteFile(ctx, fn)
	if err != nil {
		return err
	}
	_, err = connection.GetAttr(ctx, fn)
	if err != nil {
		return err
	}
	return nil
}
