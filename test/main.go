package main

import (
	"context"
	"log"
	"time"

	pb "github.com/opendedup/sdfs-client-go/sdfs"
	"google.golang.org/grpc"
)

const (
	address = "localhost:50051"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewVolumeServiceClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.GetVolumeInfo(ctx, &pb.VolumeInfoRequest{})
	if err != nil {
		log.Fatalf("could not get Volume name: %v", err)
	}
	log.Printf("Volume Name is : %s", r.GetName())
	// Make a File
	fio := pb.NewFileIOServiceClient(conn)
	var file string = "wowee2.txt"
	mkr, zerr := fio.Mknod(ctx, &pb.MkNodRequest{Path: file})
	if zerr != nil {
		log.Fatalf("could not make file: %v", zerr)
	}
	if mkr.GetErrorCode() > 0 {
		log.Printf("Error Creating File Code %s", mkr.GetErrorCode())
	} else {
		log.Printf("File Created : %s", file)
	}

	fdr, err := fio.Open(ctx, &pb.FileOpenRequest{Path: file})
	if err != nil {
		log.Fatalf("could not make file: %v", err)
	}
	if fdr.GetErrorCode() > 0 {
		log.Printf("Error Opening File Code %s", fdr.GetErrorCode())
	} else {
		log.Printf("File Opened : %s", file)
	}
	b := []byte("I like cheese\n")
	fwr, err := fio.Write(ctx, &pb.DataWriteRequest{FileHandle: fdr.GetFileHandle(),
		Data: b, Start: 0, Len: int32(len(b))})
	if err != nil {
		log.Fatalf("could not write file: %v", err)
	}
	if fwr.GetErrorCode() > 0 {
		log.Printf("Error Write File Code %s", fwr.GetErrorCode())
	} else {
		log.Printf("File Written : %s", file)
	}
	fcr, err := fio.Release(ctx, &pb.FileCloseRequest{FileHandle: fdr.GetFileHandle()})
	if err != nil {
		log.Fatalf("could not Close file: %v", err)
	}
	if fcr.GetErrorCode() > 0 {
		log.Printf("Error Close File Code %s", fcr.GetErrorCode())
	} else {
		log.Printf("File Closed : %s", file)
	}
	fir, err := fio.GetFileInfo(ctx, &pb.FileInfoRequest{FileName: file})
	if err != nil {
		log.Fatalf("could not get file: %v", err)
	}
	if fir.GetErrorCode() > 0 {
		log.Printf("Error fileinfo File Code %s", fir.GetErrorCode())
	} else {
		log.Printf("File Info Recieved : %s", file)
		log.Printf("File Size : %d", fir.GetResponse()[0].GetSize())
	}
	fsr, err := fio.Stat(ctx, &pb.FileInfoRequest{FileName: file})
	if err != nil {
		log.Fatalf("could not get file: %v", err)
	}
	if fsr.GetErrorCode() > 0 {
		log.Printf("Error fileinfo File Code %s", fsr.GetErrorCode())
	} else {
		log.Printf("File Stat Recieved : %s", file)
		log.Printf("File Size : %d", fsr.GetResponse()[0].GetSize())
	}
	fsr, err = fio.Stat(ctx, &pb.FileInfoRequest{FileName: "/"})
	if err != nil {
		log.Fatalf("could not get file: %v", err)
	}
	if fsr.GetErrorCode() > 0 {
		log.Printf("Error fileinfo File Code %s", fsr.GetErrorCode())
	} else {
		log.Printf("File Stat Recieved : %s", fsr.GetResponse()[0].GetFileName())
		log.Printf("File Size : %d", fsr.GetResponse()[0].GetSize())
	}
	fdr, err = fio.Open(ctx, &pb.FileOpenRequest{Path: file})
	if err != nil {
		log.Fatalf("could not make file: %v", err)
	}
	if fdr.GetErrorCode() > 0 {
		log.Printf("Error Opening File Code %s", fdr.GetErrorCode())
	} else {
		log.Printf("File Opened : %s", file)
	}

	frr, err := fio.Read(ctx, &pb.DataReadRequest{FileHandle: fdr.GetFileHandle(),
		Start: 0, Len: int32(fir.GetResponse()[0].GetSize())})
	if err != nil {
		log.Fatalf("could not read file: %v", err)
	}
	if frr.GetErrorCode() > 0 {
		log.Printf("Error Reading File Code %s", frr.GetErrorCode())
	} else {
		log.Printf("File Read : %s", file)
		log.Printf("%s", string(frr.GetData()))
	}
	fcr, err = fio.Release(ctx, &pb.FileCloseRequest{FileHandle: fdr.GetFileHandle()})
	if err != nil {
		log.Fatalf("could not Close file: %v", err)
	}
	if fcr.GetErrorCode() > 0 {
		log.Printf("Error Close File Code %s", fcr.GetErrorCode())
	} else {
		log.Printf("File Closed : %s", file)
	}
	fdelr, err := fio.Unlink(ctx, &pb.UnlinkRequest{Path: file})
	if err != nil {
		log.Fatalf("could not delete file: %v", err)
	}
	if fdelr.GetErrorCode() > 0 {
		log.Printf("Error deleting File Code %s", fdelr.GetErrorCode())
	} else {
		log.Printf("File deleted : %s", file)
	}

}
