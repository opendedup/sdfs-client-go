package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"strings"

	uuid "github.com/google/uuid"
	xnet "github.com/minio/minio/pkg/net"
	spb "github.com/opendedup/sdfs-client-go/sdfs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	sdfsBackend   = "sdfs"
	sdfsSeparator = "/"
)

var authtoken string
var grpcAddress string
var grpcSSL bool
var verbose bool

//SdfsConnection is the connection info
type SdfsConnection struct {
	clnt  *grpc.ClientConn
	vc    spb.VolumeServiceClient
	fc    spb.FileIOServiceClient
	evt   spb.SDFSEventServiceClient
	token string
}

// A Credentials Struct
type Credentials struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	Disabletrust bool   `json:"disable_ssl_verify"`
	set          bool
}

//UserName is a hardcoded UserName
var UserName string

//Password is a hardcoded Password
var Password string

//DisableTrust is a hardcoded DisableTrust
var DisableTrust bool

type sdfsError struct {
	err       string
	errorCode spb.ErrorCodes
}

func (e *sdfsError) Error() string {
	return fmt.Sprintf("SDFS Error %s %s", e.err, e.errorCode)
}

func clientInterceptor(

	ctx context.Context,
	method string,
	req interface{},
	reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	// Logic before invoking the invoker
	// Calls the invoker to execute RPC
	if method != "/org.opendedup.grpc.VolumeService/AuthenticateUser" {
		_ctx := metadata.AppendToOutgoingContext(ctx, "authorization", "bearer "+authtoken)
		err := invoker(_ctx, method, req, reply, cc, opts...)
		if status.Code(err) == codes.Unauthenticated {
			token, err := authenicateUser(ctx)
			if err != nil {
				return err
			}
			authtoken = token
			_ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "bearer "+authtoken)
			err = invoker(_ctx, method, req, reply, cc, opts...)
			if verbose {
				log.Printf("status code %s for method %s", status.Code(err), method)
			}
			return err

		}
		if verbose {
			log.Printf("status code %s for method %s", status.Code(err), method)
		}
		return err
	}
	err := invoker(ctx, method, req, reply, cc, opts...)
	if verbose {
		log.Printf("status code %s for method %s", status.Code(err), method)
	}
	return err

	// Logic after invoking the invoker

}

func clientStreamInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	// Logic before invoking the invoker
	// Calls the invoker to execute RPC
	_ctx := metadata.AppendToOutgoingContext(ctx, "authorization", "bearer "+authtoken)
	s, err := streamer(_ctx, desc, cc, method, opts...)
	if err != nil {
		return nil, err
	}
	return s, nil

	// Logic after invoking the invoker

}

//CloseConnection closes the grpc connection to the volume
func (n *SdfsConnection) CloseConnection(ctx context.Context) error {
	return n.clnt.Close()
}

func authenicateUser(ctx context.Context) (token string, err error) {
	username, password, disabletrust, err := getCredentials()

	if err != nil {
		return token, err
	}
	var conn *grpc.ClientConn

	if grpcSSL == true {
		config := &tls.Config{}
		if disabletrust {
			config = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
		conn, err = grpc.Dial(grpcAddress, grpc.WithBlock(), grpc.WithTransportCredentials(credentials.NewTLS(config)))

	} else {
		conn, err = grpc.Dial(grpcAddress, grpc.WithInsecure(), grpc.WithBlock())
	}

	if err != nil {
		log.Fatalf("did not connect: %v", err)
		return token, fmt.Errorf("unable to initialize sdfsClient")
	}
	vc := spb.NewVolumeServiceClient(conn)
	auth, err := vc.AuthenticateUser(ctx, &spb.AuthenticationRequest{Username: username, Password: password})
	if err != nil {
		return token, err
	} else if auth.GetErrorCode() > 0 && auth.GetErrorCode() != spb.ErrorCodes_EEXIST {
		return token, &sdfsError{err: auth.GetError(), errorCode: auth.GetErrorCode()}
	}
	token = auth.GetToken()
	if verbose {
		log.Printf("found token %s", token)
	}
	err = conn.Close()
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	return token, err
}

func getCredentials() (username string, password string, disabletrust bool, err error) {
	username = UserName
	password = Password
	disabletrust = DisableTrust

	user, err := user.Current()
	filepath := user.HomeDir + "/.sdfs/credentials.json"
	if len(username) == 0 {
		username, _ = os.LookupEnv("SDFS_USER")
	}
	if len(password) == 0 {
		password, _ = os.LookupEnv("SDFS_PASSWORD")
	}
	epath, eok := os.LookupEnv("SDFS_CREDENTIALS_PATH")
	if !disabletrust {
		_, dok := os.LookupEnv("SDFS_DISABLE_TRUST")

		if dok {
			disabletrust = true
		}
	}
	if len(username) > 0 && len(password) > 0 {
		return username, password, disabletrust, nil
	} else if eok {
		filepath = epath
	}

	jsonFile, err := os.Open(filepath)
	if err != nil {
		return username, password, disabletrust, err
	}
	// we initialize our Users array
	var creds Credentials
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return username, password, disabletrust, err
	}
	json.Unmarshal(byteValue, &creds)
	if !disabletrust && creds.Disabletrust {
		disabletrust = creds.Disabletrust
	}
	return creds.Username, creds.Password, disabletrust, nil
}

//NewConnection Created a new connectio given a path
func NewConnection(path string) (*SdfsConnection, error) {
	var address string
	var commonPath string
	var useSSL bool
	u, err := xnet.ParseURL(path)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(u.Scheme, "sdfs") {
		return nil, fmt.Errorf("unsupported scheme %s, only supports sdfs:// or sdfss://", u)
	}
	if u.Scheme == "sdfss" {
		useSSL = true
		grpcSSL = true
	}
	if commonPath == "" {
		commonPath = u.Path
	}
	address = u.Host
	grpcAddress = address

	var conn *grpc.ClientConn
	_, err = user.Current()
	if err != nil {
		return nil, fmt.Errorf("unable to lookup local user. %s", err)
	}
	_, _, disabletrust, err := getCredentials()
	if err != nil {
		return nil, fmt.Errorf("Not able to read credentials. %s", err)
	}
	if useSSL == true {
		config := &tls.Config{}
		if disabletrust {
			config = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
		conn, err = grpc.Dial(address, grpc.WithBlock(), grpc.WithUnaryInterceptor(clientInterceptor), grpc.WithStreamInterceptor(clientStreamInterceptor), grpc.WithTransportCredentials(credentials.NewTLS(config)))

	} else {
		conn, err = grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithUnaryInterceptor(clientInterceptor))
	}

	if err != nil {
		log.Fatalf("did not connect: %v", err)
		return nil, fmt.Errorf("unable to initialize sdfsClient")
	}

	vc := spb.NewVolumeServiceClient(conn)
	fc := spb.NewFileIOServiceClient(conn)
	evt := spb.NewSDFSEventServiceClient(conn)

	return &SdfsConnection{clnt: conn, vc: vc, fc: fc, evt: evt}, nil
}

func (n *SdfsConnection) sdfsPathJoin(args ...string) string {
	return path.Join(args...)
}

//RmDir removes a given directory
func (n *SdfsConnection) RmDir(ctx context.Context, path string) error {
	rc, err := n.fc.RmDir(ctx, &spb.RmDirRequest{Path: path})
	if err != nil {
		log.Print(err)
		return err
	} else if rc.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: rc.GetError(), errorCode: rc.GetErrorCode()}
	} else {
		return nil
	}
}

//MkDir makes a given directory
func (n *SdfsConnection) MkDir(ctx context.Context, path string) error {
	rc, err := n.fc.MkDir(ctx, &spb.MkDirRequest{Path: path})
	if err != nil {
		log.Print(err)
		return err
	} else if rc.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: rc.GetError(), errorCode: rc.GetErrorCode()}
	} else {
		return nil
	}
}

//MkDirAll makes a directory and all parent directories
func (n *SdfsConnection) MkDirAll(ctx context.Context, path string) error {
	rc, err := n.fc.MkDirAll(ctx, &spb.MkDirRequest{Path: path})
	if err != nil {
		log.Print(err)
		return err
	} else if rc.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: rc.GetError(), errorCode: rc.GetErrorCode()}
	} else {
		return nil
	}
}

//Stat gets a specific file info
func (n *SdfsConnection) Stat(ctx context.Context, path string) (*spb.FileInfoResponse, error) {
	fi, err := n.fc.Stat(ctx, &spb.FileInfoRequest{FileName: path})
	if err != nil {
		log.Print(err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return nil, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return fi.GetResponse()[0], nil
}

//ListDir lists a directory
func (n *SdfsConnection) ListDir(ctx context.Context, path string) ([]*spb.FileInfoResponse, error) {
	fi, err := n.fc.GetFileInfo(ctx, &spb.FileInfoRequest{FileName: path, NumberOfFiles: 1000000, Compact: false})
	if err != nil {
		log.Print(err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return nil, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return fi.GetResponse(), nil
}

//DeleteFile removes a given file
func (n *SdfsConnection) DeleteFile(ctx context.Context, path string) error {
	fi, err := n.fc.Unlink(ctx, &spb.UnlinkRequest{Path: path})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//CopyFile creates a snapshop of a give source to a given destination
func (n *SdfsConnection) CopyFile(ctx context.Context, src, dst string, returnImmediately bool) (event *spb.SDFSEvent, err error) {
	fi, err := n.fc.CreateCopy(ctx, &spb.FileSnapshotRequest{
		Src:  src,
		Dest: dst,
	})
	if err != nil {
		log.Print(err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return nil, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	eventid := fi.EventID
	if returnImmediately {
		return n.GetEvent(ctx, eventid)
	}
	return n.WaitForEvent(ctx, eventid)
}

//GetEvent returns the event struct for a specific event id
func (n *SdfsConnection) GetEvent(ctx context.Context, eventid string) (*spb.SDFSEvent, error) {
	fi, err := n.evt.GetEvent(ctx, &spb.SDFSEventRequest{Uuid: eventid})
	if err != nil {
		log.Printf("unable to get id %s, error: %v \n", eventid, err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		return nil, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return fi.Event, nil
}

//ListEvents lists all the events that have occured
func (n *SdfsConnection) ListEvents(ctx context.Context) ([]*spb.SDFSEvent, error) {
	fi, err := n.evt.ListEvents(ctx, &spb.SDFSEventListRequest{})
	if err != nil {
		log.Printf("unable to list events %v \n", err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return nil, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return fi.Events, nil
}

//WaitForEvent waits for events to finish and then returns the end event.
func (n *SdfsConnection) WaitForEvent(ctx context.Context, eventid string) (*spb.SDFSEvent, error) {
	stream, err := n.evt.SubscribeEvent(ctx, &spb.SDFSEventRequest{Uuid: eventid})
	if err != nil {

		log.Print(err)
		return nil, err
	}
	for {
		fi, err := stream.Recv()
		if err == io.EOF {
			return n.GetEvent(ctx, eventid)
		}
		if err != nil {

			log.Print(err)
			return nil, err
		} else if fi.GetErrorCode() > 0 {
			log.Print(err)
			return nil, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
		}
		if fi.Event.EndTime > 0 {
			return fi.Event, nil
		}
	}
}

//GetXAttrSize gets the list size for attributes. This is useful for a fuse implementation
func (n *SdfsConnection) GetXAttrSize(ctx context.Context, path string) (int32, error) {
	fi, err := n.fc.GetXAttrSize(ctx, &spb.GetXAttrSizeRequest{Path: path})
	if err != nil {
		log.Print(err)
		return 0, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return 0, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return fi.Length, nil
}

//Fsync Syncs an sdfs file to underlying storage
func (n *SdfsConnection) Fsync(ctx context.Context, path string, fh int64) error {
	fi, err := n.fc.Fsync(ctx, &spb.FsyncRequest{Path: path, Fh: fh})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//SetXAttr sets a specific key value pair for a given file
func (n *SdfsConnection) SetXAttr(ctx context.Context, key, value, path string) error {
	fi, err := n.fc.SetXAttr(ctx, &spb.SetXAttrRequest{Path: path, Attr: key, Value: value})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//RemoveXAttr removes a given key for a file
func (n *SdfsConnection) RemoveXAttr(ctx context.Context, key, path string) error {
	fi, err := n.fc.RemoveXAttr(ctx, &spb.RemoveXAttrRequest{Path: path, Attr: key})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//GetXAttr retrieves a value for a given attribute and file
func (n *SdfsConnection) GetXAttr(ctx context.Context, key, path string) (value string, err error) {
	fi, err := n.fc.GetXAttr(ctx, &spb.GetXAttrRequest{Path: path, Attr: key})
	if err != nil {
		log.Print(err)
		return value, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return value, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return fi.Value, nil
}

//Utime sets the utime for a given file
func (n *SdfsConnection) Utime(ctx context.Context, path string, atime, mtime int64) (err error) {
	fi, err := n.fc.Utime(ctx, &spb.UtimeRequest{Path: path, Atime: atime, Mtime: mtime})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//Truncate truncates a given file to a given length in bytes
func (n *SdfsConnection) Truncate(ctx context.Context, path string, length int64) (err error) {
	fi, err := n.fc.Truncate(ctx, &spb.TruncateRequest{Path: path, Length: length})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//SymLink creates a symlink for a given source and destination
func (n *SdfsConnection) SymLink(ctx context.Context, src, dst string) (err error) {
	fi, err := n.fc.SymLink(ctx, &spb.SymLinkRequest{From: src, To: dst})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//ReadLink creates a symlink for a given source and destination
func (n *SdfsConnection) ReadLink(ctx context.Context, path string) (linkpath string, err error) {
	fi, err := n.fc.ReadLink(ctx, &spb.LinkRequest{Path: path})
	if err != nil {
		log.Print(err)
		return linkpath, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return linkpath, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return fi.LinkPath, nil
}

//GetAttr returns Stat for a given file
func (n *SdfsConnection) GetAttr(ctx context.Context, path string) (stat *spb.Stat, err error) {
	fi, err := n.fc.GetAttr(ctx, &spb.StatRequest{Path: path})
	if err != nil {
		log.Print(err)
		return stat, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return stat, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return fi.Stat, nil
}

//Flush flushes the requested file to underlying storage
func (n *SdfsConnection) Flush(ctx context.Context, path string, fh int64) (err error) {
	fi, err := n.fc.Flush(ctx, &spb.FlushRequest{Path: path, Fd: fh})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//Chown sets ownership of the requested file to the requested owner
func (n *SdfsConnection) Chown(ctx context.Context, path string, gid int32, uid int32) (err error) {
	fi, err := n.fc.Chown(ctx, &spb.ChownRequest{Path: path, Gid: gid, Uid: uid})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//Chmod sets permission of the requested file to the requested permissions
func (n *SdfsConnection) Chmod(ctx context.Context, path string, mode int32) (err error) {
	fi, err := n.fc.Chmod(ctx, &spb.ChmodRequest{Path: path, Mode: mode})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//Unlink deletes the given file
func (n *SdfsConnection) Unlink(ctx context.Context, path string) (err error) {
	fi, err := n.fc.Unlink(ctx, &spb.UnlinkRequest{Path: path})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//Write writes data to a given filehandle
func (n *SdfsConnection) Write(ctx context.Context, fh int64, data []byte, offset int64, length int32) (err error) {
	fi, err := n.fc.Write(ctx, &spb.DataWriteRequest{FileHandle: fh, Data: data, Len: length})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//Read reads data from a given filehandle
func (n *SdfsConnection) Read(ctx context.Context, fh int64, offset int64, length int32) (data []byte, err error) {
	fi, err := n.fc.Read(ctx, &spb.DataReadRequest{FileHandle: fh, Start: offset, Len: length})
	if err != nil {
		log.Print(err)
		return data, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return data, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return fi.Data, nil
}

//Release closes a given filehandle
func (n *SdfsConnection) Release(ctx context.Context, fh int64) (err error) {
	fi, err := n.fc.Release(ctx, &spb.FileCloseRequest{FileHandle: fh})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//MkNod makes a given file
func (n *SdfsConnection) MkNod(ctx context.Context, path string) (err error) {
	fi, err := n.fc.Mknod(ctx, &spb.MkNodRequest{Path: path})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//Open opens a given file
func (n *SdfsConnection) Open(ctx context.Context, path string) (fh int64, err error) {
	fi, err := n.fc.Open(ctx, &spb.FileOpenRequest{Path: path})
	if err != nil {
		log.Print(err)
		return fh, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return fh, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return fi.FileHandle, nil
}

//FileExists checks if a file Exists given a path.
func (n *SdfsConnection) FileExists(ctx context.Context, path string) (exists bool, err error) {
	fi, err := n.fc.FileExists(ctx, &spb.FileExistsRequest{Path: path})
	if err != nil {
		log.Print(err)
		return false, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return false, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return fi.Exists, nil
}

//SetUserMetaData sets an array of key value pairs for a given path
func (n *SdfsConnection) SetUserMetaData(ctx context.Context, path string, fileAttributes []*spb.FileAttributes) (err error) {
	fi, err := n.fc.SetUserMetaData(ctx, &spb.SetUserMetaDataRequest{Path: path, FileAttributes: fileAttributes})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//GetCloudFile hydrates a given file from object storage to the local filesystem.
func (n *SdfsConnection) GetCloudFile(ctx context.Context, path, dst string, overwrite, waitForCompletion bool) (event *spb.SDFSEvent, err error) {
	fi, err := n.fc.GetCloudFile(ctx, &spb.GetCloudFileRequest{File: path, Dstfile: dst, Overwrite: overwrite, Changeid: uuid.New().String()})
	if err != nil {
		log.Print(err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return nil, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	eventid := fi.EventID
	if waitForCompletion {
		return n.WaitForEvent(ctx, eventid)
	}
	return n.GetEvent(ctx, eventid)
}

//GetCloudMetaFile downloads the metadata for given file from object storage but does not hydrate it into the local hashtable.
func (n *SdfsConnection) GetCloudMetaFile(ctx context.Context, path, dst string, overwrite, waitForCompletion bool) (event *spb.SDFSEvent, err error) {
	fi, err := n.fc.GetCloudMetaFile(ctx, &spb.GetCloudFileRequest{File: path, Dstfile: dst, Overwrite: overwrite, Changeid: uuid.New().String()})
	if err != nil {
		log.Print(err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return nil, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	eventid := fi.EventID
	if waitForCompletion {
		return n.WaitForEvent(ctx, eventid)
	}
	return n.GetEvent(ctx, eventid)
}

//GetVolumeInfo returns the volume info
func (n *SdfsConnection) GetVolumeInfo(ctx context.Context) (volumeInfo *spb.VolumeInfoResponse, err error) {
	fi, err := n.vc.GetVolumeInfo(ctx, &spb.VolumeInfoRequest{})
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return fi, nil

}

//ShutdownVolume unmounts the given volume
func (n *SdfsConnection) ShutdownVolume(ctx context.Context) (err error) {
	_, err = n.vc.ShutdownVolume(ctx, &spb.ShutdownRequest{})
	if err != nil {
		//log.Print(err)
		return err
	}
	return nil
}

//CleanStore does garbage collection on the volume
func (n *SdfsConnection) CleanStore(ctx context.Context, compact, waitForCompletion bool) (event *spb.SDFSEvent, err error) {
	fi, err := n.vc.CleanStore(ctx, &spb.CleanStoreRequest{Compact: compact})
	if err != nil {
		log.Print(err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return nil, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	eventid := fi.EventID
	if waitForCompletion {
		return n.WaitForEvent(ctx, eventid)
	}
	return n.GetEvent(ctx, eventid)
}

//DeleteCloudVolume deletes a volume that is no longer in use
func (n *SdfsConnection) DeleteCloudVolume(ctx context.Context, volumeid int64, waitForCompletion bool) (event *spb.SDFSEvent, err error) {
	fi, err := n.vc.DeleteCloudVolume(ctx, &spb.DeleteCloudVolumeRequest{Volumeid: volumeid})
	if err != nil {
		log.Print(err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return nil, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	eventid := fi.EventID
	if waitForCompletion {
		return n.WaitForEvent(ctx, eventid)
	}
	return n.GetEvent(ctx, eventid)
}

//DSEInfo get dedupe storage info
func (n *SdfsConnection) DSEInfo(ctx context.Context) (info *spb.DSEInfo, err error) {
	fi, err := n.vc.DSEInfo(ctx, &spb.DSERequest{})
	if err != nil {
		log.Print(err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return nil, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return fi.Info, nil
}

//SystemInfo returns system info
func (n *SdfsConnection) SystemInfo(ctx context.Context) (info *spb.SystemInfo, err error) {
	fi, err := n.vc.SystemInfo(ctx, &spb.SystemInfoRequest{})
	if err != nil {
		log.Print(err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return nil, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return fi.Info, nil
}

//SetVolumeCapacity sets the logical size of the volume
func (n *SdfsConnection) SetVolumeCapacity(ctx context.Context, size int64) (err error) {
	fi, err := n.vc.SetVolumeCapacity(ctx, &spb.SetVolumeCapacityRequest{Size: size})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//GetConnectedVolumes returns all the volumes sharing the same storage
func (n *SdfsConnection) GetConnectedVolumes(ctx context.Context) (info []*spb.ConnectedVolumeInfo, err error) {
	fi, err := n.vc.GetConnectedVolumes(ctx, &spb.CloudVolumesRequest{})
	if err != nil {
		log.Print(err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return nil, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return fi.VolumeInfo, nil
}

//GetGCSchedule returns the current Garbage collection schedule for the volume
func (n *SdfsConnection) GetGCSchedule(ctx context.Context) (schedule string, err error) {
	fi, err := n.vc.GetGCSchedule(ctx, &spb.GCScheduleRequest{})
	if err != nil {
		log.Print(err)
		return schedule, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return schedule, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return fi.Schedule, nil
}

//SetCacheSize does garbage collection on the volume
func (n *SdfsConnection) SetCacheSize(ctx context.Context, size int64, waitForCompletion bool) (event *spb.SDFSEvent, err error) {
	fi, err := n.vc.SetCacheSize(ctx, &spb.SetCacheSizeRequest{CacheSize: size})
	if err != nil {
		log.Print(err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return nil, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	eventid := fi.EventID
	if waitForCompletion {
		return n.WaitForEvent(ctx, eventid)
	}
	return n.GetEvent(ctx, eventid)
}

//SetPassword sets the password for the volume
func (n *SdfsConnection) SetPassword(ctx context.Context, password string) (err error) {
	fi, err := n.vc.SetPassword(ctx, &spb.SetPasswordRequest{Password: password})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//SetReadSpeed sets the read speed in Kb/s
func (n *SdfsConnection) SetReadSpeed(ctx context.Context, speed int32) (err error) {
	fi, err := n.vc.SetReadSpeed(ctx, &spb.SpeedRequest{RequestedSpeed: speed})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//SetWriteSpeed sets the write speed in Kb/s
func (n *SdfsConnection) SetWriteSpeed(ctx context.Context, speed int32) (err error) {
	fi, err := n.vc.SetReadSpeed(ctx, &spb.SpeedRequest{RequestedSpeed: speed})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	return nil
}

//SyncFromCloudVolume syncs the current volume from a give volume id
func (n *SdfsConnection) SyncFromCloudVolume(ctx context.Context, volumeid int64, waitForCompletion bool) (event *spb.SDFSEvent, err error) {
	fi, err := n.vc.SyncFromCloudVolume(ctx, &spb.SyncFromVolRequest{Volumeid: volumeid})
	if err != nil {
		log.Print(err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return nil, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	eventid := fi.EventID
	if waitForCompletion {
		return n.WaitForEvent(ctx, eventid)
	}
	return n.GetEvent(ctx, eventid)
}

//SyncCloudVolume syncs the current volume from all instances in the cloud
func (n *SdfsConnection) SyncCloudVolume(ctx context.Context, waitForCompletion bool) (event *spb.SDFSEvent, err error) {
	fi, err := n.vc.SyncCloudVolume(ctx, &spb.SyncVolRequest{})
	if err != nil {
		log.Print(err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		log.Print(err)
		return nil, &sdfsError{err: fi.GetError(), errorCode: fi.GetErrorCode()}
	}
	eventid := fi.EventID
	if waitForCompletion {
		return n.WaitForEvent(ctx, eventid)
	}
	return n.GetEvent(ctx, eventid)
}
