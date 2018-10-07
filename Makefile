GOOS=linux
GOARCH=amd64

EXECUTABLE=chat-bot-hub
RUNTIME_PATH=build
RUNTIME_IMAGE=chat-bot-hub
PACKAGE=github.com/hawkwithwind/$(EXECUTABLE)

GOIMAGE=golang:1.11-alpine3.8

SOURCES=$(shell find . -type f -name '*.go' -not -path "./vendor/*")
BASE=$(GOPATH)/src/$(PACKAGE)

#build-runtime: $(RUNTIME_PATH)/$(EXECUTABLE) $(RUNTIME_PATH)/dockerfile
#	docker build -t $(EXECUTABLE) $(RUNTIME_PATH)

$(RUNTIME_PATH)/$(EXECUTABLE): $(SOURCES) $(RUNTIME_PATH)
	docker run --rm \
	-v $(GOPATH)/src:/go/src \
	-v $(GOPATH)/pkg:/go/pkg \
	-v $(shell pwd)/$(RUNTIME_PATH):/go/bin/${GOOS}_${GOARCH} \
	-e GOOS=$(GOOS) -e GOARCH=$(GOARCH) -e GOBIN=/go/bin/$(GOOS)_$(GOARCH) -e CGO_ENABLED=0 \
	$(GOIMAGE) go get -a -installsuffix cgo $(PACKAGE)/...

#$(RUNTIME_PATH)/dockerfile: runtime-image
#	sh make_docker_file.sh $(RUNTIME_PATH)/Dockerfile $(RUNTIME_IMAGE) $(EXECUTABLE)

#runtime-image:
#	docker build -t $(RUNTIME_IMAGE) docker/runtime

$(RUNTIME_PATH):
	[ -d $(RUNTIME_PATH) ] || mkdir $(RUNTIME_PATH)

clean:
	rm -rf $(RUNTIME_PATH)

fmt:
	docker run --rm \
	-v $(shell pwd):/go/src/$(PACKAGE) \
	$(GOIMAGE) sh -c "cd /go/src/$(PACKAGE)" && gofmt -l -w $(SOURCES)


