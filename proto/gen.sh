go get -u github.com/golang/protobuf/protoc-gen-go
go get -u github.com/favadi/protoc-go-inject-tag

protoc -I chatbothub/ chatbothub/chatbothub.proto --go_out=plugins=grpc:chatbothub
protoc-go-inject-tag -input=chatbothub/chatbothub.pb.go

protoc -I streaming/ streaming/streaming.proto --go_out=plugins=grpc:streaming