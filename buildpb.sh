export PATH=$PATH:$HOME/go/bin/
cd ..
git clone https://github.com/grpc/grpc-go.git
cd grpc-go/cmd/protoc-gen-go-grpc
go install .
protoc -I ../sdfs/src/proto/ ../sdfs/src/proto/IOService.proto --go_out=sdfs --go-grpc_out=sdfs --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative
git tag <tag>
git push origin <tag>
