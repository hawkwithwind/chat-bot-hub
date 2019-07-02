go get -u github.com/golang/protobuf/protoc-gen-go
go get -u github.com/favadi/protoc-go-inject-tag

protoc -I chatbothub/ chatbothub/chatbothub.proto --go_out=plugins=grpc:chatbothub
protoc-go-inject-tag -input=chatbothub/chatbothub.pb.go

protoc -I web/ web/web.proto --go_out=plugins=grpc:web
protoc-go-inject-tag -input=web/web.pb.go
