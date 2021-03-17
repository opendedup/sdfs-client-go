package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"strings"
	"time"

	uuid "github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	xnet "github.com/minio/minio/pkg/net"
	spb "github.com/opendedup/sdfs-client-go/sdfs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"
)

const (
	sdfsBackend    = "sdfs"
	sdfsSeparator  = "/"
	sdfsTempFolder = ".sdfsclitemp"
)

const slashSeparator = "/"

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
	ServerURL    string `yaml:"url" required:"true" envconfig:"SDFS_URL" default:"sdfss://localhost:6442"`
	Password     string `yaml:"password" default:"" envconfig:"SDFS_PASSWORD" default:""`
	DisableTrust bool   `yaml:"disable_trust" envconfig:"SDFS_DISABLE_TRUST"`
	set          bool
}

//UserName is a hardcoded UserName
var UserName string

//Password is a hardcoded Password
var Password string

//DisableTrust is a hardcoded DisableTrust
var DisableTrust bool

//SdfsError is an SDFS Error with an error code mapped as a syscall error id
type SdfsError struct {
	Err       string
	ErrorCode spb.ErrorCodes
}

func (e *SdfsError) Error() string {
	return fmt.Sprintf("SDFS Error %s %s", e.Err, e.ErrorCode)
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
				log.Printf("unauthenticated status code %s for method %s", status.Code(err), method)
			}
			return err

		}
		if verbose && err != nil {
			log.Printf("authenticated status code %s for method %s", status.Code(err), method)
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
	creds, err := getCredentials("")

	if err != nil {
		return token, err
	}
	var conn *grpc.ClientConn

	if grpcSSL == true {
		config := &tls.Config{}
		if creds.DisableTrust {
			config = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
		conn, err = grpc.Dial(grpcAddress, grpc.WithBlock(), grpc.WithTransportCredentials(credentials.NewTLS(config)))

	} else {
		conn, err = grpc.Dial(grpcAddress, grpc.WithInsecure(), grpc.WithBlock())
	}

	if err != nil {
		log.Printf("did not connect: %v", err)
		return token, fmt.Errorf("unable to initialize sdfsClient")
	}
	vc := spb.NewVolumeServiceClient(conn)
	auth, err := vc.AuthenticateUser(ctx, &spb.AuthenticationRequest{Username: "admin", Password: creds.Password})
	if err != nil {
		log.Printf("did not connect: %v", err)
		return token, err
	} else if auth.GetErrorCode() > 0 && auth.GetErrorCode() != spb.ErrorCodes_EEXIST {
		log.Printf("did not connect: %v", auth)
		return token, &SdfsError{Err: auth.GetError(), ErrorCode: auth.GetErrorCode()}
	}
	token = auth.GetToken()
	if verbose {
		log.Printf("found token %s", token)
	}
	err = conn.Close()
	if err != nil {
		log.Printf("did not connect: %v", err)
	}
	return token, err
}

func getCredentials(configPath string) (creds *Credentials, err error) {
	if configPath == "" {
		user, err := user.Current()
		if err != nil {
			return nil, err
		}
		configPath = user.HomeDir + "/.sdfs/credentials.yaml"
	}
	// Create config structure
	creds = &Credentials{}

	// Init environmental variables
	err = envconfig.Process("", creds)
	if err != nil {
		return nil, err
	}
	_, err = os.Stat(configPath)
	if os.IsNotExist(err) {
		if DisableTrust {
			creds.DisableTrust = true
		}

		if len(Password) > 0 {
			creds.Password = Password
		}
		return creds, nil
	}
	log.Printf("Reading Credentials from %s \n", configPath)
	// Open config file
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	// Init new YAML decode
	d := yaml.NewDecoder(file)

	// Start YAML decoding from file
	if err := d.Decode(&creds); err != nil {
		return nil, err
	}
	creds.ServerURL = strings.ToLower(creds.ServerURL)
	if DisableTrust {
		creds.DisableTrust = true
	}

	if !strings.HasPrefix(creds.ServerURL, "sdfs") {
		return nil, fmt.Errorf("unsupported server type %s, only supports sdfs:// or sdfss://", creds.ServerURL)
	}
	if len(Password) > 0 {
		creds.Password = Password
	}
	return creds, nil

}

func parseURL(url string) (string, string, bool, error) {
	u, err := xnet.ParseURL(url)
	if err != nil {
		return "", "", false, err
	}
	if !strings.HasPrefix(u.Scheme, "sdfs") {
		return "", "", false, fmt.Errorf("unsupported scheme %s, only supports sdfs:// or sdfss://", u)
	}
	useSSL := false
	commonPath := ""
	if u.Scheme == "sdfss" {
		useSSL = true
	} else if commonPath == "" {
		commonPath = u.Path
	}
	address := u.Host
	return address, commonPath, useSSL, nil
}

//AddTrustedCert adds a trusted Cert
func AddTrustedCert(url string) error {
	address, _, _, err := parseURL(url)
	if err != nil {
		return err
	}
	config := tls.Config{InsecureSkipVerify: true}
	conn, err := tls.Dial("tcp", address, &config)

	if err != nil {
		return err
	}
	defer conn.Close()
	var b bytes.Buffer
	for _, cert := range conn.ConnectionState().PeerCertificates {
		err := pem.Encode(&b, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
		if err != nil {
			return err
		}
	}
	user, err := user.Current()
	if err != nil {
		return err
	}
	configPath := user.HomeDir + "/.sdfs/keys/"
	os.MkdirAll(configPath, 0700)
	fn := fmt.Sprintf("%s%s.pem", configPath, address)
	err = ioutil.WriteFile(fn, b.Bytes(), 0600)
	if err != nil {
		return err
	}
	nb, err := ioutil.ReadFile(fn)
	if err != nil {
		return err
	}
	block, _ := pem.Decode(nb)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}
	//fmt.Printf("cert=%v", cert)

	certPool := x509.NewCertPool()
	certPool.AddCert(cert)
	nconfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         cert.Subject.CommonName,
		RootCAs:            certPool,
	}
	nconn, err := tls.Dial("tcp", address, nconfig)
	if err != nil {
		return err
	}
	nconn.Close()
	return nil
}

func getCert(address string) (*x509.Certificate, error) {
	user, err := user.Current()
	if err != nil {
		return nil, err
	}
	configPath := user.HomeDir + "/.sdfs/keys/"

	fn := fmt.Sprintf("%s%s.pem", configPath, address)
	_, err = os.Stat(fn)
	if os.IsNotExist(err) {
		return nil, nil
	}
	nb, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(nb)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
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
	} else if commonPath == "" {
		commonPath = u.Path
	}
	address = u.Host
	grpcAddress = address

	var conn *grpc.ClientConn
	_, err = user.Current()
	if err != nil {
		return nil, fmt.Errorf("unable to lookup local user. %s", err)
	}

	creds, err := getCredentials("")
	creds.ServerURL = path
	if err != nil {
		log.Printf("Not able to read credentials. %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5000*time.Millisecond)
	defer cancel()
	if useSSL == true {
		config := &tls.Config{}
		if creds.DisableTrust {
			config = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
		cert, err := getCert(address)
		if err != nil {
			return nil, err
		} else if cert != nil {
			certPool := x509.NewCertPool()
			certPool.AddCert(cert)
			config = &tls.Config{
				InsecureSkipVerify: false,
				ServerName:         cert.Subject.CommonName,
				RootCAs:            certPool,
			}
		}

		//fmt.Printf("TLS Connecting to %s  disable_trust=%t\n", address, creds.DisableTrust)
		conn, err = grpc.DialContext(ctx, address, grpc.WithBlock(), grpc.WithUnaryInterceptor(clientInterceptor), grpc.WithStreamInterceptor(clientStreamInterceptor), grpc.WithTransportCredentials(credentials.NewTLS(config)))

	} else {
		//fmt.Printf("Connecting to %s \n", address)
		conn, err = grpc.DialContext(ctx, address, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithUnaryInterceptor(clientInterceptor), grpc.WithStreamInterceptor(clientStreamInterceptor))
	}
	//fmt.Print("BLA")

	if err != nil {
		log.Printf("did not connect to %s : %v\n", path, err)
		return nil, fmt.Errorf("unable to initialize sdfsClient")
	}
	if conn == nil {
		return nil, fmt.Errorf("unable to initialize sdfsClient")
	}

	vc := spb.NewVolumeServiceClient(conn)
	_, err = vc.GetGCSchedule(ctx, &spb.GCScheduleRequest{})
	if err != nil {
		return nil, err
	}
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
		return &SdfsError{Err: rc.GetError(), ErrorCode: rc.GetErrorCode()}
	} else {
		return nil
	}
}

//MkDir makes a given directory
func (n *SdfsConnection) MkDir(ctx context.Context, path string, mode int32) error {
	rc, err := n.fc.MkDir(ctx, &spb.MkDirRequest{Path: path, Mode: mode})
	if err != nil {
		log.Print(err)
		return err
	} else if rc.GetErrorCode() > 0 {
		return &SdfsError{Err: rc.GetError(), ErrorCode: rc.GetErrorCode()}
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
		return &SdfsError{Err: rc.GetError(), ErrorCode: rc.GetErrorCode()}
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
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	return fi.GetResponse()[0], nil
}

//ListDir lists a directory
func (n *SdfsConnection) ListDir(ctx context.Context, path, marker string, compact bool, returnsz int32) (string, []*spb.FileInfoResponse, error) {
	fi, err := n.fc.GetFileInfo(ctx, &spb.FileInfoRequest{FileName: path, NumberOfFiles: returnsz, Compact: false, ListGuid: marker})
	if err != nil {
		log.Print(err)
		return "", nil, err
	} else if fi.GetErrorCode() > 0 {
		return "", nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	return fi.ListGuid, fi.GetResponse(), nil
}

//DeleteFile removes a given file
func (n *SdfsConnection) DeleteFile(ctx context.Context, path string) error {
	fi, err := n.fc.Unlink(ctx, &spb.UnlinkRequest{Path: path})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	return nil
}

//CopyExtent creates a snapshop of a give source to a given destination
func (n *SdfsConnection) CopyExtent(ctx context.Context, src, dst string, srcStart, dstStart, len int64) (written int64, err error) {
	fi, err := n.fc.CopyExtent(ctx, &spb.CopyExtentRequest{SrcFile: src, DstFile: dst, SrcStart: srcStart, DstStart: dstStart, Length: len})
	if err != nil {
		log.Print(err)
		return 0, err
	} else if fi.GetErrorCode() > 0 {
		return 0, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	return fi.Written, nil
}

//StatFS Gets Filesystem in Typical OS Stat Format
func (n *SdfsConnection) StatFS(ctx context.Context) (stat *spb.StatFS, err error) {
	fi, err := n.fc.StatFS(ctx, &spb.StatFSRequest{})
	if err != nil {
		log.Print(err)
		return stat, err
	} else if fi.GetErrorCode() > 0 {
		return stat, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	return fi.Stat, nil
}

//Rename renames a file
func (n *SdfsConnection) Rename(ctx context.Context, src, dst string) (err error) {
	fi, err := n.fc.Rename(ctx, &spb.FileRenameRequest{Src: src, Dest: dst})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
			return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
		} else if fi.Event.Level == "error" {
			return nil, &SdfsError{Err: fi.Event.ShortMsg, ErrorCode: spb.ErrorCodes_EBADFD}
		}
		if fi.Event.EndTime > 0 {
			return fi.Event, nil
		}
	}
}

//FileNotification notifies over a channel for all events when a file is downloaded for Sync
func (n *SdfsConnection) FileNotification(ctx context.Context, fileInfo chan *spb.FileMessageResponse) error {
	stream, err := n.fc.FileNotification(ctx, &spb.SyncNotificationSubscription{Uid: uuid.New().String()})
	if err != nil {

		log.Print(err)
		return err
	}
	for {
		fi, err := stream.Recv()
		if err != nil {

			log.Print(err)
			fileInfo <- nil
			return err
		} else if fi.GetErrorCode() > 0 {
			return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
		} else {
			if len(fi.GetResponse()) > 0 {
				fileInfo <- fi
			}

		}
	}
}

func (n *SdfsConnection) SetMaxAge(ctx context.Context, maxAge int64) error {
	fi, err := n.vc.SetMaxAge(ctx, &spb.SetMaxAgeRequest{MaxAge: maxAge})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	return nil
}

//GetXAttrSize gets the list size for attributes. This is useful for a fuse implementation
func (n *SdfsConnection) GetXAttrSize(ctx context.Context, path, key string) (int32, error) {
	fi, err := n.fc.GetXAttrSize(ctx, &spb.GetXAttrSizeRequest{Path: path, Attr: key})
	if err != nil {
		log.Print(err)
		return 0, err
	} else if fi.GetErrorCode() > 0 {
		return 0, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return value, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	return nil
}

//ReadLink reads a symlink for a given source
func (n *SdfsConnection) ReadLink(ctx context.Context, path string) (linkpath string, err error) {
	fi, err := n.fc.ReadLink(ctx, &spb.LinkRequest{Path: path})
	if err != nil {
		log.Print(err)
		return linkpath, err
	} else if fi.GetErrorCode() > 0 {
		return linkpath, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return stat, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	return nil
}

//Write writes data to a given filehandle
func (n *SdfsConnection) Write(ctx context.Context, fh int64, data []byte, offset int64, length int32) (err error) {
	fi, err := n.fc.Write(ctx, &spb.DataWriteRequest{FileHandle: fh, Data: data, Len: length, Start: offset})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return data, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	return nil
}

//MkNod makes a given file
func (n *SdfsConnection) MkNod(ctx context.Context, path string, mode, rdev int32) (err error) {
	fi, err := n.fc.Mknod(ctx, &spb.MkNodRequest{Path: path, Mode: mode, Rdev: rdev})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	return nil
}

//Open opens a given file
func (n *SdfsConnection) Open(ctx context.Context, path string, flags int32) (fh int64, err error) {
	fi, err := n.fc.Open(ctx, &spb.FileOpenRequest{Path: path, Flags: flags})
	if err != nil {
		log.Print(err)
		return fh, err
	} else if fi.GetErrorCode() > 0 {
		return fh, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return false, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	eventid := fi.EventID
	if waitForCompletion {
		return n.WaitForEvent(ctx, eventid)
	}
	return n.GetEvent(ctx, eventid)
}

//Clear AuthToken will remove the current auth token to authenticate to the client
func ClearAuthToken() {
	authtoken = ""
}

//DeleteCloudVolume deletes a volume that is no longer in use
func (n *SdfsConnection) DeleteCloudVolume(ctx context.Context, volumeid int64, waitForCompletion bool) (event *spb.SDFSEvent, err error) {
	fi, err := n.vc.DeleteCloudVolume(ctx, &spb.DeleteCloudVolumeRequest{Volumeid: volumeid})
	if err != nil {
		log.Print(err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return schedule, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	eventid := fi.EventID
	_, err = n.WaitForEvent(ctx, eventid)
	return err
}

//SetWriteSpeed sets the write speed in Kb/s
func (n *SdfsConnection) SetWriteSpeed(ctx context.Context, speed int32) (err error) {
	fi, err := n.vc.SetWriteSpeed(ctx, &spb.SpeedRequest{RequestedSpeed: speed})
	if err != nil {
		log.Print(err)
		return err
	} else if fi.GetErrorCode() > 0 {
		return &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	eventid := fi.EventID
	_, err = n.WaitForEvent(ctx, eventid)
	return err
}

//SyncFromCloudVolume syncs the current volume from a give volume id
func (n *SdfsConnection) SyncFromCloudVolume(ctx context.Context, volumeid int64, waitForCompletion bool) (event *spb.SDFSEvent, err error) {
	fi, err := n.vc.SyncFromCloudVolume(ctx, &spb.SyncFromVolRequest{Volumeid: volumeid})
	if err != nil {
		log.Print(err)
		return nil, err
	} else if fi.GetErrorCode() > 0 {
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
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
		return nil, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	eventid := fi.EventID
	if waitForCompletion {
		return n.WaitForEvent(ctx, eventid)
	}
	return n.GetEvent(ctx, eventid)
}

//Upload uploads a file to the filesystem
func (n *SdfsConnection) Upload(ctx context.Context, src, dst string) (written int64, err error) {
	u, err := uuid.NewRandom()
	if err != nil {
		return -1, err
	}
	info, err := os.Stat(src)
	if err != nil {
		return -1, err
	}
	if info.IsDir() {
		return -1, fmt.Errorf(" %s is a dir", src)
	}
	tmpname := path.Join(sdfsTempFolder, u.String())
	var fh int64
	n.fc.MkDirAll(ctx, &spb.MkDirRequest{Path: sdfsTempFolder})
	mkf, err := n.fc.Mknod(ctx, &spb.MkNodRequest{Path: tmpname})
	if err != nil {
		return -1, err
	} else if mkf.GetErrorCode() > 0 {
		return -1, &SdfsError{Err: mkf.GetError(), ErrorCode: mkf.GetErrorCode()}
	}
	fhr, err := n.fc.Open(ctx, &spb.FileOpenRequest{Path: tmpname})
	if err != nil {
		return -1, err
	} else if fhr.GetErrorCode() > 0 {
		return -1, &SdfsError{Err: fhr.GetError(), ErrorCode: fhr.GetErrorCode()}
	}
	defer n.fc.Unlink(ctx, &spb.UnlinkRequest{Path: tmpname})
	fh = fhr.GetFileHandle()
	b1 := make([]byte, 128*1024)
	var offset int64 = 0
	var n1 int = 0
	r, err := os.Open(src)
	defer r.Close()
	if err != nil {
		return -1, err
	}
	n1, err = r.Read(b1)
	s := make([]byte, n1)
	copy(s, b1)
	fwr, err := n.fc.Write(ctx, &spb.DataWriteRequest{FileHandle: fh, Data: s, Start: offset, Len: int32(n1)})
	offset += int64(n1)
	if err != nil {
		n.fc.Release(ctx, &spb.FileCloseRequest{FileHandle: fh})
		return -1, err
	} else if fwr.GetErrorCode() > 0 {
		n.fc.Release(ctx, &spb.FileCloseRequest{FileHandle: fh})
		return -1, &SdfsError{Err: fwr.GetError(), ErrorCode: fwr.GetErrorCode()}
	}
	for n1 > 0 {
		n1, err = r.Read(b1)
		if n1 > 0 {
			s = make([]byte, n1)
			copy(s, b1)
			fwr, err = n.fc.Write(ctx, &spb.DataWriteRequest{FileHandle: fh, Data: s, Start: offset, Len: int32(n1)})
			offset += int64(n1)
			if err != nil {
				n.fc.Release(ctx, &spb.FileCloseRequest{FileHandle: fh})
				return -1, err
			} else if fwr.GetErrorCode() > 0 {
				n.fc.Release(ctx, &spb.FileCloseRequest{FileHandle: fh})
				return -1, &SdfsError{Err: fwr.GetError(), ErrorCode: fwr.GetErrorCode()}
			}
		}
	}

	n.fc.Release(ctx, &spb.FileCloseRequest{FileHandle: fh})
	dir := path.Dir(dst)
	if dir != "" {
		mkd, err := n.fc.MkDirAll(ctx, &spb.MkDirRequest{Path: dir})
		if err != nil {
			return -1, err
		} else if mkd.GetErrorCode() > 0 && mkd.GetErrorCode() != spb.ErrorCodes_EEXIST {
			return -1, &SdfsError{Err: mkd.GetError(), ErrorCode: mkd.GetErrorCode()}
		}
	}
	n.fc.Unlink(ctx, &spb.UnlinkRequest{Path: dst})

	sp, err := n.fc.Rename(ctx, &spb.FileRenameRequest{Src: tmpname, Dest: dst})
	if err != nil {
		return -1, err
	} else if sp.GetErrorCode() > 0 {
		return -1, &SdfsError{Err: sp.GetError(), ErrorCode: sp.GetErrorCode()}
	}

	fi, err := n.fc.Stat(ctx, &spb.FileInfoRequest{FileName: dst})
	if err != nil {
		return -1, err
	} else if fi.GetErrorCode() > 0 {
		return -1, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}

	return fi.GetResponse()[0].GetSize(), nil
}

//Download downloads a file from SDFS locally
func (n *SdfsConnection) Download(ctx context.Context, src, dst string) (bytesread int64, err error) {

	fi, err := n.fc.Stat(ctx, &spb.FileInfoRequest{FileName: src})
	if err != nil {
		return -1, err
	} else if fi.GetErrorCode() > 0 {
		return -1, &SdfsError{Err: fi.GetError(), ErrorCode: fi.GetErrorCode()}
	}
	rd, err := n.fc.Open(ctx, &spb.FileOpenRequest{Path: src})
	if err != nil {
		return -1, err
	} else if rd.GetErrorCode() > 0 {
		return -1, &SdfsError{Err: rd.GetError(), ErrorCode: rd.GetErrorCode()}
	}
	defer n.fc.Release(ctx, &spb.FileCloseRequest{FileHandle: rd.GetFileHandle()})
	var read int64 = 0
	var blocksize int32 = 128 * 1024
	var length = fi.GetResponse()[0].Size
	writer, err := os.Create(dst)
	defer writer.Close()
	for read < length {
		if blocksize > int32(length-read) {
			blocksize = int32(length - read)
		}
		if blocksize == 0 {
			break
		}
		//log(" reading at  %d len %d\n", read, blocksize)
		rdr, err := n.fc.Read(ctx, &spb.DataReadRequest{FileHandle: rd.GetFileHandle(), Len: blocksize, Start: read})
		if err != nil {

			return -1, err
		} else if rdr.GetErrorCode() > 0 {
			return -1, &SdfsError{Err: rdr.GetError(), ErrorCode: rdr.GetErrorCode()}
		}
		_, err = writer.Write(rdr.GetData())
		if err != nil {
			return -1, err
		}
		read += int64(blocksize)
	}

	return read, nil
}
