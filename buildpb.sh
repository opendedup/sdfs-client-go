protoc -I ../sdfs/src/proto/ ../sdfs/src/proto/IOService.proto --go_out=sdfs --go-grpc_out=sdfs --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative