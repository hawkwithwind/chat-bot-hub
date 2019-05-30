go get -u github.com/golang/protobuf/protoc-gen-go

protoc -I chatbothub/ chatbothub/chatbothub.proto --go_out=plugins=grpc:chatbothub
protoc -I streaming/ streaming/streaming.proto --go_out=plugins=grpc:streaming