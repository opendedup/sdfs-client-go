package dedupe

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	b64 "encoding/base64"

	"github.com/ReneKroon/ttlcache/v2"
	"github.com/ahmetask/worker"
	lru "github.com/hashicorp/golang-lru"
	rabin "github.com/opendedup/go-rabin/rabin"
	spb "github.com/opendedup/sdfs-client-go/sdfs"
	"github.com/pierrec/lz4/v4"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
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
	fileHandles map[string]*DedupeFile
	connection  *grpc.ClientConn
	chunkSize   int64
	hashType    HashType
	minLen      int64
	maxLen      int64
	hc          spb.StorageServiceClient
	rabinTable  *rabin.Table
	pool        *worker.Pool
	mu          sync.Mutex
	bufferSize  int
	Compress    bool
	ddcache     *ttlcache.Cache
	pVolumeID   int64
}

type DedupeBuffer struct {
	buffer     []byte
	mu         sync.Mutex
	offset     int64
	fileName   string
	fileHandle int64
	limit      int32
	Flushed    bool
	Flushing   bool
}

type DedupeFile struct {
	mu              sync.Mutex
	flushMu         sync.RWMutex
	cache           *lru.Cache
	flushingBuffers map[int64]*DedupeBuffer
	fileHandles     map[int64]bool
	fileName        string
	err             error
	pVolumeID       int64
}

type Job struct {
	buffer *DedupeBuffer
	engine *DedupeEngine
	file   *DedupeFile
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

func CompressData(data []byte) (cdata []byte, err error) {
	var c lz4.Compressor
	cdata = make([]byte, lz4.CompressBlockBound(len(data)))

	len, err := c.CompressBlock(data, cdata)
	if err != nil {
		log.Error(err)
		return cdata, err
	}
	return cdata[:len], nil
}

func DecompressData(data []byte, len int32) (ddata []byte, err error) {
	ddata = make([]byte, len)
	_, err = lz4.UncompressBlock(data, ddata)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return ddata, nil
}

func getGUID(fh, volumeID int64) string {
	return fmt.Sprintf("%d-%d", volumeID, fh)
}

func getFileGuid(file string, volumeID int64) string {
	return fmt.Sprintf("%s-%d", file, volumeID)
}

func NewDedupeEngine(ctx context.Context, connection *grpc.ClientConn, size, threads int, debug bool, compressed bool, volumeid int64, cacheSize int, cacheDuration int) (*DedupeEngine, error) {
	log.Out = os.Stdout
	if debug {
		log.SetLevel(logrus.DebugLevel)
	}
	pool := worker.NewWorkerPool(threads, 1)

	//Start worker pool
	pool.Start()
	hc := spb.NewStorageServiceClient(connection)
	fi, err := hc.HashingInfo(ctx, &spb.HashingInfoRequest{PvolumeID: volumeid})
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
	log.Debugf("hashing info %v", fi)
	dd := &DedupeEngine{
		openFiles:   make(map[string]*DedupeFile),
		fileHandles: make(map[string]*DedupeFile),
		hc:          hc,
		connection:  connection,
		chunkSize:   fi.ChunkSize,
		hashType:    hashType,
		minLen:      fi.MinSegmentSize,
		maxLen:      fi.MaxSegmentSize,
		rabinTable:  rabin.NewTable(uint64(fi.PolyNumber), int(fi.WindowSize)),
		pool:        pool,
		pVolumeID:   volumeid,
		bufferSize:  size,
		Compress:    compressed,
		ddcache:     ttlcache.NewCache(),
	}
	dd.ddcache.SetTTL(time.Duration(time.Duration(cacheDuration) * time.Minute))
	dd.ddcache.SetCacheSizeLimit(cacheSize)

	return dd, nil
}

func (n *DedupeEngine) HashingInfo(ctx context.Context) (*spb.HashingInfoResponse, error) {
	fi, err := n.hc.HashingInfo(ctx, &spb.HashingInfoRequest{PvolumeID: n.pVolumeID})
	if err != nil {
		log.Print(err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	return fi, nil
}

func (n *DedupeEngine) Open(fileName string, fileHandle int64, volumeID int64) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	file, ok := n.openFiles[getFileGuid(fileName, volumeID)]
	if !ok {

		file = &DedupeFile{
			fileHandles:     make(map[int64]bool),
			fileName:        fileName,
			flushingBuffers: make(map[int64]*DedupeBuffer),
			pVolumeID:       volumeID,
		}
		onEvicted := func(k interface{}, v interface{}) {
			buffer := v.(*DedupeBuffer)
			file.flushMu.Lock()
			file.flushingBuffers[k.(int64)] = v.(*DedupeBuffer)
			file.flushMu.Unlock()
			buffer.Flushing = true
			n.pool.Submit(&Job{buffer: buffer, engine: n, file: file})

		}
		l, err := lru.NewWithEvict(n.bufferSize, onEvicted)
		if err != nil {
			log.Errorf("unable initialize %d : %v", fileHandle, err)
			return err
		}
		file.cache = l

		n.openFiles[getFileGuid(fileName, volumeID)] = file
	}
	file.mu.Lock()
	file.fileHandles[fileHandle] = true
	file.mu.Unlock()
	n.fileHandles[getGUID(fileHandle, volumeID)] = file
	return nil
}

func (n *DedupeEngine) Sync(fileHandle int64, volumeID int64) error {
	n.mu.Lock()
	file, ok := n.fileHandles[getGUID(fileHandle, volumeID)]
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
					file.flushMu.Lock()
					file.flushingBuffers[k.(int64)] = v.(*DedupeBuffer)
					file.flushMu.Unlock()
					buffer.Flushing = true
					wg.Add(1)
					n.pool.Submit(&Job{buffer: buffer, engine: n, wg: &wg, file: file})
				}
			}
			wg.Wait()
		}
		if file.err != nil {
			log.Errorf("error during Previous Write IO Operation detected %v", file.err)
			return fmt.Errorf("error during Previous Write IO Operation detected %v", file.err)
		}
	}
	return nil
}

func (n *DedupeEngine) Close(fileHandle int64, volumeID int64) error {
	n.Sync(fileHandle, volumeID)

	n.mu.Lock()
	defer n.mu.Unlock()
	file, ok := n.fileHandles[getGUID(fileHandle, volumeID)]

	if ok {
		delete(n.fileHandles, getGUID(fileHandle, volumeID))
		file.mu.Lock()
		defer file.mu.Unlock()
		delete(file.fileHandles, fileHandle)
		m := len(file.fileHandles)
		if m == 0 {
			delete(n.openFiles, getFileGuid(file.fileName, volumeID))
		}
		if file.err != nil {
			log.Errorf("error during Previous Write IO Operation detected %v", file.err)
			return fmt.Errorf("error during Previous Write IO Operation detected %v", file.err)
		}
	}

	return nil
}

func (n *DedupeEngine) CloseFile(fileName string, volumeID int64) error {
	n.SyncFile(fileName, volumeID)
	n.mu.Lock()
	defer n.mu.Unlock()
	file, ok := n.openFiles[getFileGuid(fileName, volumeID)]

	if ok {
		file.mu.Lock()
		defer file.mu.Unlock()
		for k, _ := range file.fileHandles {
			delete(n.fileHandles, getGUID(k, file.pVolumeID))
		}
		delete(n.openFiles, file.fileName)
		if file.err != nil {
			log.Errorf("error during Previous Write IO Operation detected %v", file.err)
			return fmt.Errorf("error during Previous Write IO Operation detected %v", file.err)
		}

	}

	return nil

}

func (n *DedupeEngine) SyncFile(fileName string, volumeID int64) error {
	n.mu.Lock()
	file, ok := n.openFiles[getFileGuid(fileName, volumeID)]

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
					file.flushMu.Lock()
					file.flushingBuffers[k.(int64)] = v.(*DedupeBuffer)
					file.flushMu.Unlock()
					buffer.Flushing = true
					wg.Add(1)
					n.pool.Submit(&Job{buffer: buffer, engine: n, wg: &wg, file: file})
				}
			}
			wg.Wait()
		}
		if file.err != nil {
			log.Errorf("error during Previous Write IO Operation detected %v", file.err)
			return fmt.Errorf("error during Previous Write IO Operation detected %v", file.err)
		}
	}
	return nil
}

var notFound = ttlcache.ErrNotFound

func (n *DedupeEngine) CheckHashes(ctx context.Context, fingers []*Finger, volumeID int64) ([]*Finger, error) {
	log.Debug("in checking hashes")
	defer log.Debug("done checking hashes")
	chreq := &spb.CheckHashesRequest{PvolumeID: volumeID}
	hes := make([][]byte, 0)
	hm := make(map[string][]int)
	for i := 0; i < len(fingers); i++ {
		sEnc := b64.StdEncoding.EncodeToString(fingers[i].hash)
		if val, err := n.ddcache.Get(getFileGuid(sEnc, volumeID)); err != notFound {
			fingers[i].archive = val.(int64)
			fingers[i].dedup = true
		} else {

			val, ok := hm[sEnc]
			if !ok {
				val = make([]int, 0)
			}
			val = append(val, i)
			hm[sEnc] = val
			hes = append(hes, fingers[i].hash)
		}

	}
	chreq.Hashes = hes
	if len(hes) > 0 {
		fi, err := n.hc.CheckHashes(ctx, chreq)
		if err != nil {
			log.Errorf("error cheching hashes %v", err)
			return nil, err
		} else if fi.GetErrorCode() > 0 {
			log.Errorf("error checking hashes error : %s , error code: %s", fi.Error, fi.ErrorCode)
			return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
		}

		log.Debugf("check hashes %d", len(hes))

		for i := 0; i < len(fi.Locations); i++ {
			hs := hes[i]
			sEnc := b64.StdEncoding.EncodeToString(hs)
			val, ok := hm[sEnc]
			if !ok {
				return nil, fmt.Errorf("unable to find %s", sEnc)
			}
			for _, s := range val {
				fingers[s].archive = fi.Locations[i]
				if fingers[i].archive != -1 {
					fingers[i].dedup = true
					loc := fingers[s].archive
					n.ddcache.Set(getFileGuid(sEnc, volumeID), loc)
				}
			}
		}
	}
	return fingers, nil
}

func (n *DedupeEngine) WriteChunks(ctx context.Context, fingers []*Finger, fileHandle int64, volumeID int64) ([]*Finger, error) {
	log.Debug("in write chunks")
	defer log.Debug("done writing chunks")
	ces := make([]*spb.ChunkEntry, 0)
	hl := make([]int, 0)
	wchreq := &spb.WriteChunksRequest{FileHandle: fileHandle, PvolumeID: volumeID}
	for i := 0; i < len(fingers); i++ {
		if !fingers[i].dedup {
			if n.Compress && len(fingers[i].data) > 10 {
				buf, err := CompressData(fingers[i].data)
				if err != nil {
					log.Errorf("error compressing chunks %v", err)
				}
				if len(buf) > len(fingers[i].data) || err != nil {
					ce := &spb.ChunkEntry{Hash: fingers[i].hash, Data: fingers[i].data, Compressed: false}
					ces = append(ces, ce)
				} else {
					ce := &spb.ChunkEntry{Hash: fingers[i].hash, Data: buf, Compressed: true, CompressedLength: int32(len(fingers[i].data))}
					ces = append(ces, ce)
				}

			} else {
				ce := &spb.ChunkEntry{Hash: fingers[i].hash, Data: fingers[i].data, Compressed: false}
				ces = append(ces, ce)
			}
			hl = append(hl, i)
		}
		if len(fingers[i].data) == 0 {
			log.Warnf("found null data at %d arlen %d", i, len(fingers))
		}
	}
	if len(ces) > 0 {
		wchreq.Chunks = ces
		fi, err := n.hc.WriteChunks(ctx, wchreq)
		if err != nil {
			log.Errorf("error during writechunk api call %v", err)
			return nil, err
		} else if fi.GetErrorCode() > 0 {
			log.Errorf("error writing chunks error : %s , error code: %s", fi.Error, fi.ErrorCode)
			return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
		}
		for i := 0; i < len(fi.InsertRecords); i++ {
			z := hl[i]
			if !fingers[z].dedup {
				fingers[z].archive = fi.InsertRecords[i].Hashloc
				fingers[z].dedup = !fi.InsertRecords[i].Inserted
				if fingers[z].archive == -1 && len(fingers[z].data) > 0 {
					log.Warnf("Archive should not be -1")
					return nil, fmt.Errorf("archive should not be -1")
				}
			}
		}
	}
	return fingers, nil
}

func (n *DedupeEngine) WriteSparseDataChunk(ctx context.Context, fingers []*Finger, fileHandle, fileLocation int64, length int32, volumeID int64) error {
	log.Debug("in write sparse chunks")
	defer log.Debug("done writing sparse chunks")
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
	var sr *spb.SparseDedupeChunkWriteRequest
	if n.Compress {
		out, err := proto.Marshal(sdc)
		if err != nil {
			log.Errorf("error during sparsedata chunk marshalling %v", err)
			return err
		}
		chunk, err := CompressData(out)
		if err != nil {
			log.Errorf("error out = %d, %v", len(out), err)
			sr = &spb.SparseDedupeChunkWriteRequest{
				Chunk:        sdc,
				FileHandle:   fileHandle,
				FileLocation: fileLocation,
				PvolumeID:    volumeID,
				Compressed:   false,
			}
		} else {
			sr = &spb.SparseDedupeChunkWriteRequest{
				FileHandle:      fileHandle,
				FileLocation:    fileLocation,
				PvolumeID:       volumeID,
				Compressed:      true,
				CompressedChunk: chunk,
				UncompressedLen: int32(len(out)),
			}
		}

		log.Debugf("compressed sdc from %d to %d", len(out), len(chunk))

	} else {
		sr = &spb.SparseDedupeChunkWriteRequest{Chunk: sdc, FileHandle: fileHandle, FileLocation: fileLocation, PvolumeID: volumeID}
	}
	log.Debugf("writing sdc %d at position %d", len(pairs), sr.FileLocation)
	fi, err := n.hc.WriteSparseDataChunk(ctx, sr)
	if err != nil {
		log.Errorf("error while writing chunk %v", err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Errorf("error writing sparse chunks error : %s , error code: %s", fi.Error, fi.ErrorCode)
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	//log.Debugf("wrote %v", sr)
	return nil

}

func (n *DedupeEngine) Write(fileHandle, offset int64, wbuffer []byte, length int32, volumeID int64) error {
	log.Debugf("Writing at offset %d len %d", offset, length)
	file, ok := n.fileHandles[getGUID(fileHandle, volumeID)]
	if !ok {
		log.Errorf("filehandle not found %s", getGUID(fileHandle, volumeID))
		return fmt.Errorf("filehandle not found %s", getGUID(fileHandle, volumeID))
	}
	if file.err != nil {
		log.Errorf("error during Previous Write IO Operation detected %v", file.err)
		return fmt.Errorf("error during Previous Write IO Operation detected %v", file.err)
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
		file.mu.Lock()
		val, ok := file.cache.Peek(fpos)
		if !ok {
			log.Debugf("creating %d", fpos)
			val, ok = file.cache.Peek(fpos)
			if !ok {
				file.flushMu.RLock()
				val, ok = file.flushingBuffers[fpos]
				file.flushMu.RUnlock()
				if ok {
					file.flushMu.Lock()
					delete(file.flushingBuffers, fpos)
					file.flushMu.Unlock()
					chunk := val.(*DedupeBuffer)
					chunk.Flushing = false
					chunk.Flushed = false
					file.cache.Add(fpos, val)
				}

			}
			if !ok {
				buf := make([]byte, n.chunkSize)
				val = &DedupeBuffer{offset: fpos, buffer: buf, fileName: file.fileName}
				file.cache.Add(fpos, val)
			}

		}
		file.mu.Unlock()
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
		file.mu.Lock()
		file.cache.Add(fpos, val)
		file.mu.Unlock()
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
	if !j.buffer.Flushed && j.buffer.Flushing {
		var fingers []*Finger
		log.Debugf("bytes = [%d] [%d] [%d]", len(j.buffer.buffer), j.buffer.offset, j.buffer.limit)
		r := bytes.NewReader(j.buffer.buffer[:j.buffer.limit])

		c := rabin.NewChunker(j.engine.rabinTable, r, int(j.engine.minLen), int(j.engine.maxLen))
		var nextPos int32
		for i := 0; ; i++ {
			clen, err := c.Next()
			if err == io.EOF || clen == 0 {
				break
			} else if err != nil {
				log.Errorf("error while in job %v", err)
				j.engine.mu.Lock()
				file, ok := j.engine.openFiles[getFileGuid(j.buffer.fileName, j.file.pVolumeID)]
				j.engine.mu.Unlock()
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
			//log.Debugf("hash is %s length is %d", hex.EncodeToString(finger.hash), clen)
			fingers = append(fingers, finger)
			nextPos += int32(clen)
		}
		fingers, err := j.engine.CheckHashes(ctx, fingers, j.file.pVolumeID)
		if err != nil {
			log.Errorf("error while checking hashes %v", err)
			j.engine.mu.Lock()
			file, ok := j.engine.openFiles[getFileGuid(j.buffer.fileName, j.file.pVolumeID)]
			j.engine.mu.Unlock()
			if ok {
				file.err = err
			}
			return
		}

		fingers, err = j.engine.WriteChunks(ctx, fingers, j.buffer.fileHandle, j.file.pVolumeID)
		if err != nil {
			log.Errorf("error while writing chunks %v", err)
			j.engine.mu.Lock()
			file, ok := j.engine.openFiles[getFileGuid(j.buffer.fileName, j.file.pVolumeID)]
			j.engine.mu.Unlock()
			if ok {
				file.err = err
			}
			return
		}
		err = j.engine.WriteSparseDataChunk(ctx, fingers, j.buffer.fileHandle, j.buffer.offset, j.buffer.limit, j.file.pVolumeID)
		if err != nil {
			log.Errorf("error while writing sparse chunks %v", err)
			log.Error(err)
			j.engine.mu.Lock()
			file, ok := j.engine.openFiles[getFileGuid(j.buffer.fileName, j.file.pVolumeID)]
			j.engine.mu.Unlock()
			if ok {
				file.err = err
			}
			return
		}
		j.file.flushMu.Lock()
		delete(j.file.flushingBuffers, j.buffer.offset)
		j.file.flushMu.Unlock()
		j.buffer.Flushing = false
		j.buffer.Flushed = true
	}

}
