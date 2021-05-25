package dedupe

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/ahmetask/worker"
	lru "github.com/hashicorp/golang-lru"
	rabin "github.com/opendedup/go-rabin/rabin"
	spb "github.com/opendedup/sdfs-client-go/sdfs"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type HashType int

var log = logrus.New()

const (
	MD5 HashType = iota
	SHA265
	SHA256_160
)

type DedupeEngine struct {
	openFiles   map[string]*DedupeFile
	fileHandles map[int64]*DedupeFile
	connection  *grpc.ClientConn
	chunkSize   int64
	hashType    HashType
	minLen      int64
	maxLen      int64
	hc          spb.StorageServiceClient
	rabinTable  *rabin.Table
	pool        *worker.Pool
	mu          sync.Mutex
}

type DedupeBuffer struct {
	buffer     []byte
	mu         sync.Mutex
	offset     int64
	fileName   string
	fileHandle int64
	limit      int32
	Flushed    bool
}

type DedupeFile struct {
	mu          sync.Mutex
	cache       *lru.Cache
	fileHandles map[int64]bool
	fileName    string
	err         error
}

type Job struct {
	buffer *DedupeBuffer
	engine *DedupeEngine
	wg     *sync.WaitGroup
}

//SdfsError is an SDFS Error with an error code mapped as a syscall error id
type SdfsError struct {
	Err       string
	ErrorCode spb.ErrorCodes
}

func (e *SdfsError) Error() string {
	return fmt.Sprintf("SDFS Error %s %s", e.Err, e.ErrorCode)
}

func NewDedupeEngine(ctx context.Context, connection *grpc.ClientConn, size, threads int, debug bool) (*DedupeEngine, error) {
	log.Out = os.Stdout
	if debug {
		log.SetLevel(logrus.DebugLevel)
	}
	pool := worker.NewWorkerPool(1, 1)

	//Start worker pool
	pool.Start()
	hc := spb.NewStorageServiceClient(connection)
	fi, err := hc.HashingInfo(ctx, &spb.HashingInfoRequest{})
	if err != nil {
		log.Errorf("unable to get hashinginfo %v", err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		log.Errorf("unable to get hashinginfo %v %s", fi.ErrorCode, fi.Error)
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	hashType := SHA265
	if fi.Hashtype == spb.Hashtype_MD5 {
		hashType = MD5
	} else if fi.Hashtype == spb.Hashtype_UNSUPPORTED {
		return nil, fmt.Errorf("unsupported hashtype")
	}
	log.Debugf("hashhing info %v", fi)
	return &DedupeEngine{
		openFiles:   make(map[string]*DedupeFile),
		fileHandles: make(map[int64]*DedupeFile),
		hc:          hc,
		connection:  connection,
		chunkSize:   fi.ChunkSize,
		hashType:    hashType,
		minLen:      fi.MinSegmentSize,
		maxLen:      fi.MaxSegmentSize,
		rabinTable:  rabin.NewTable(uint64(fi.PolyNumber), int(fi.WindowSize)),
		pool:        pool,
	}, nil
}

func (n *DedupeEngine) HashingInfo(ctx context.Context) (*spb.HashingInfoResponse, error) {
	fi, err := n.hc.HashingInfo(ctx, &spb.HashingInfoRequest{})
	if err != nil {
		log.Print(err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	return fi, nil
}

func (n *DedupeEngine) Open(fileName string, fileHandle int64) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	onEvicted := func(k interface{}, v interface{}) {
		buffer := v.(*DedupeBuffer)
		n.pool.Submit(&Job{buffer: buffer, engine: n})

	}
	file, ok := n.openFiles[fileName]
	if !ok {
		l, err := lru.NewWithEvict(4, onEvicted)
		if err != nil {
			log.Errorf("unable initialize %d : %v", fileHandle, err)
			return err
		}
		file = &DedupeFile{
			cache:       l,
			fileHandles: make(map[int64]bool),
			fileName:    fileName,
		}
		n.openFiles[fileName] = file
	}
	file.mu.Lock()
	file.fileHandles[fileHandle] = true
	file.mu.Unlock()
	n.fileHandles[fileHandle] = file
	return nil
}

func (n *DedupeEngine) Sync(fileHandle int64) {
	n.mu.Lock()
	file, ok := n.fileHandles[fileHandle]
	n.mu.Unlock()
	if ok {
		file.mu.Lock()
		defer file.mu.Unlock()
		m := len(file.fileHandles)

		if m > 0 {
			var wg sync.WaitGroup
			for _, k := range file.cache.Keys() {
				v, ok := file.cache.Get(k)
				if ok {
					buffer := v.(*DedupeBuffer)
					wg.Add(1)
					n.pool.Submit(&Job{buffer: buffer, engine: n, wg: &wg})
				}
			}
			wg.Wait()
		}
	}
}

func (n *DedupeEngine) Close(fileHandle int64) {
	n.Sync(fileHandle)
	n.mu.Lock()
	defer n.mu.Unlock()
	file, ok := n.fileHandles[fileHandle]
	if ok {
		delete(n.fileHandles, fileHandle)
		file.mu.Lock()
		defer file.mu.Unlock()
		delete(file.fileHandles, fileHandle)
		m := len(file.fileHandles)
		if m == 0 {
			delete(n.openFiles, file.fileName)
		}
	}
}

func (n *DedupeEngine) CloseFile(fileName string) {
	n.SyncFile(fileName)
	n.mu.Lock()
	defer n.mu.Unlock()
	file, ok := n.openFiles[fileName]
	if ok {
		file.mu.Lock()
		defer file.mu.Unlock()
		for k, _ := range file.fileHandles {
			delete(n.fileHandles, k)
		}
		delete(n.openFiles, file.fileName)

	}

}

func (n *DedupeEngine) SyncFile(fileName string) {
	n.mu.Lock()
	file, ok := n.openFiles[fileName]
	n.mu.Unlock()

	if ok {
		file.mu.Lock()
		defer file.mu.Unlock()

		m := len(file.fileHandles)

		if m > 0 {
			var wg sync.WaitGroup
			for _, k := range file.cache.Keys() {
				v, ok := file.cache.Get(k)
				if ok {
					buffer := v.(*DedupeBuffer)
					wg.Add(1)
					n.pool.Submit(&Job{buffer: buffer, engine: n, wg: &wg})
				}
			}
			wg.Wait()
		}
	}
}

func (n *DedupeEngine) CheckHashes(ctx context.Context, fingers []*Finger) ([]*Finger, error) {
	chreq := &spb.CheckHashesRequest{}
	hes := make([][]byte, len(fingers))
	for i := 0; i < len(fingers); i++ {
		hes[i] = fingers[i].hash
	}
	chreq.Hashes = hes
	fi, err := n.hc.CheckHashes(ctx, chreq)
	if err != nil {
		log.Errorf("error cheching hashes %v", err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		log.Errorf("error checking hashes error : %s , error code: %s", fi.Error, fi.ErrorCode)
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	log.Debugf("check hashes %v", fi)
	for i := 0; i < len(fi.Locations); i++ {
		fingers[i].archive = fi.Locations[i]
		if fingers[i].archive != -1 {
			fingers[i].dedup = true
		}
	}
	return fingers, nil
}

func (n *DedupeEngine) WriteChunks(ctx context.Context, fingers []*Finger, fileHandle int64) ([]*Finger, error) {
	ces := make([]*spb.ChunkEntry, len(fingers))
	wchreq := &spb.WriteChunksRequest{FileHandle: fileHandle}
	for i := 0; i < len(fingers); i++ {
		if !fingers[i].dedup {
			ce := &spb.ChunkEntry{Hash: fingers[i].hash, Data: fingers[i].data}
			ces[i] = ce
		} else {
			ce := &spb.ChunkEntry{}
			ces[i] = ce
		}

	}
	wchreq.Chunks = ces
	fi, err := n.hc.WriteChunks(ctx, wchreq)
	if err != nil {
		log.Error(err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		log.Errorf("error writing chunks error : %s , error code: %s", fi.Error, fi.ErrorCode)
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	for i := 0; i < len(fi.InsertRecords); i++ {
		if !fingers[i].dedup {
			fingers[i].archive = fi.InsertRecords[i].Hashloc
			fingers[i].dedup = !fi.InsertRecords[i].Inserted
			if fingers[i].archive == -1 {
				log.Warnf("Archive should not be -1")
				return nil, fmt.Errorf("archive should not be -1")
			}
		}
	}
	return fingers, nil
}

func (n *DedupeEngine) WriteSparseDataChunk(ctx context.Context, fingers []*Finger, fileHandle, fileLocation int64, length int32) error {
	pairs := make(map[int32]*spb.HashLocPairP)
	var dup int32
	for i := 0; i < len(fingers); i++ {
		pairs[fingers[i].start] = &spb.HashLocPairP{
			Hash:     fingers[i].hash,
			Hashloc:  fingers[i].archive,
			Len:      int32(fingers[i].len),
			Pos:      fingers[i].start,
			Dup:      fingers[i].dedup,
			Inserted: !fingers[i].dedup,
			Nlen:     int32(fingers[i].len),
			Offset:   0,
		}
		if fingers[i].dedup {
			dup += int32(fingers[i].len)
		}
	}
	sdc := &spb.SparseDataChunkP{Fpos: fileLocation, Len: length, Ar: pairs, Doop: dup}
	sr := &spb.SparseDedupeChunkWriteRequest{Chunk: sdc, FileHandle: fileHandle, FileLocation: fileLocation}
	log.Debugf("writing %v", sr)
	fi, err := n.hc.WriteSparseDataChunk(ctx, sr)
	if err != nil {
		log.Error(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Errorf("error writing sparse chunks error : %s , error code: %s", fi.Error, fi.ErrorCode)
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	log.Debugf("wrote %v", sr)
	return nil

}

func (n *DedupeEngine) Write(fileHandle, offset int64, wbuffer []byte, length int32) error {
	log.Debugf("Writing at offset %d len %d", offset, length)
	file, ok := n.fileHandles[fileHandle]
	if !ok {
		log.Errorf("filehandle not found %d", fileHandle)
		return fmt.Errorf("filehandle not found %d", fileHandle)
	}
	var bytesWritten int32
	for bytesWritten < length {
		noffset := offset + int64(bytesWritten)
		log.Debugf("new offset = %d", noffset)
		fpos := n.getOffset(noffset)
		bspos := int32(noffset - fpos)
		bepos := length - bytesWritten
		if bepos+bspos > int32(n.chunkSize) {
			bepos = int32(n.chunkSize) - bspos
		}
		val, ok := file.cache.Peek(fpos)
		if !ok {
			log.Debugf("creating %d", fpos)
			file.mu.Lock()
			val, ok = file.cache.Peek(fpos)
			if !ok {
				buf := make([]byte, n.chunkSize)
				val = &DedupeBuffer{offset: fpos, buffer: buf, fileName: file.fileName}
			}
			file.mu.Unlock()
		}
		chunk := val.(*DedupeBuffer)
		chunk.mu.Lock()
		chunk.fileHandle = fileHandle
		ep := bspos + bepos
		sep := bytesWritten + bepos
		w := copy(chunk.buffer[bspos:ep], wbuffer[bytesWritten:sep])
		log.Debugf("%d %d", w, len(chunk.buffer))
		chunk.Flushed = false
		chunk.mu.Unlock()

		log.Debugf("wrote fpos %d at %d len %d written %d", fpos, bspos, bepos, w)
		file.cache.Add(fpos, chunk)
		bytesWritten += int32(w)
		log.Debugf("bytes written %d", bytesWritten)
		if bspos+bepos > chunk.limit {
			chunk.limit = bspos + bepos
		}
	}
	return nil

}

func (n *DedupeEngine) getOffset(position int64) int64 {
	npos := position / n.chunkSize
	pos := npos * n.chunkSize
	return pos
}

func (n *DedupeEngine) Shutdown() {

}

type Finger struct {
	hash    []byte
	data    []byte
	start   int32
	len     int
	dedup   bool
	archive int64
}

func (j *Job) Do() {
	ctx, cancel := context.WithCancel(context.Background())
	if j.wg != nil {
		defer j.wg.Done()
	}

	defer cancel()
	j.buffer.mu.Lock()
	defer j.buffer.mu.Unlock()
	if !j.buffer.Flushed {
		var fingers []*Finger
		log.Debugf("bytes = [%d] [%d] [%d]", len(j.buffer.buffer), j.buffer.offset, j.buffer.limit)
		r := bytes.NewReader(j.buffer.buffer[:j.buffer.limit])

		c := rabin.NewChunker(j.engine.rabinTable, r, int(j.engine.minLen), int(j.engine.maxLen))
		var nextPos int32
		for i := 0; ; i++ {
			clen, err := c.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Error(err)
				file, ok := j.engine.openFiles[j.buffer.fileName]
				if ok {
					file.err = err
				}
				return
			}
			er := nextPos + int32(clen)
			finger := &Finger{data: j.buffer.buffer[nextPos:er], start: nextPos, len: clen}
			if j.engine.hashType == MD5 {
				hash16 := md5.Sum(finger.data)
				finger.hash = hash16[:]
			} else {
				hash32 := sha256.Sum256(finger.data)
				finger.hash = hash32[:]
			}
			log.Debugf("hash is %s length is %d", hex.EncodeToString(finger.hash), clen)
			fingers = append(fingers, finger)
			nextPos += int32(clen)
		}
		fingers, err := j.engine.CheckHashes(ctx, fingers)
		if err != nil {
			log.Error(err)
			file, ok := j.engine.openFiles[j.buffer.fileName]
			if ok {
				file.err = err
			}
			return
		}

		fingers, err = j.engine.WriteChunks(ctx, fingers, j.buffer.fileHandle)
		if err != nil {
			log.Error(err)
			file, ok := j.engine.openFiles[j.buffer.fileName]
			if ok {
				file.err = err
			}
			return
		}
		err = j.engine.WriteSparseDataChunk(ctx, fingers, j.buffer.fileHandle, j.buffer.offset, j.buffer.limit)
		if err != nil {
			log.Error(err)
			file, ok := j.engine.openFiles[j.buffer.fileName]
			if ok {
				file.err = err
			}
			return
		}
		j.buffer.Flushed = true
	}

}
