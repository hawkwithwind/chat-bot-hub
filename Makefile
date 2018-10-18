GOOS=linux
GOARCH=amd64

EXECUTABLE=chat-bot-hub
RUNTIME_PATH=build
RUNTIME_IMAGE=chat-bot-hub
PACKAGE=github.com/hawkwithwind/$(EXECUTABLE)

GOIMAGE=golang:1.11-alpine3.8

SOURCES=$(shell find server -type f -name '*.go' -not -path "./vendor/*")
BASE=$(GOPATH)/src/$(PACKAGE)

build-angular: $(RUNTIME_PATH)/$(EXECUTABLE)
	if [ -d $(RUNTIME_PATH)/static/ ]; then chmod -R +w $(RUNTIME_PATH)/static/ ; fi && \
	cd frontend && \
	gulp p && \
	cd .. && \
	cp -R frontend/static/img $(RUNTIME_PATH)/static/ && \
	cp -R frontend/static/lib $(RUNTIME_PATH)/static/ && \
	cp frontend/index.html $(RUNTIME_PATH)/static/ && \
	chmod -R -w $(RUNTIME_PATH)/static/

$(RUNTIME_PATH)/$(EXECUTABLE): $(SOURCES) $(RUNTIME_PATH) build-image
	docker run --rm \
	-v $(GOPATH)/src:/go/src \
	-v $(GOPATH)/pkg:/go/pkg \
	-v $(shell pwd)/$(RUNTIME_PATH):/go/bin/${GOOS}_${GOARCH} \
	-e GOOS=$(GOOS) -e GOARCH=$(GOARCH) -e GOBIN=/go/bin/$(GOOS)_$(GOARCH) -e CGO_ENABLED=0 \
	golang:withgit go get -a -installsuffix cgo $(PACKAGE)/server/...

#$(RUNTIME_PATH)/dockerfile: runtime-image
#	sh make_docker_file.sh $(RUNTIME_PATH)/Dockerfile $(RUNTIME_IMAGE) $(EXECUTABLE)

#runtime-image:
#	docker build -t $(RUNTIME_IMAGE) docker/runtime

build-image:
	docker build -t golang:withgit docker/build

$(RUNTIME_PATH):
	[ -d $(RUNTIME_PATH) ] || mkdir $(RUNTIME_PATH)

clean:
	if [ -d $(RUNTIME_PATH)/static/ ]; then chmod -R +w $(RUNTIME_PATH)/static/ ; fi && \
	rm -rf $(RUNTIME_PATH)

fmt:
	docker run --rm \
	-v $(shell pwd):/go/src/$(PACKAGE) \
	$(GOIMAGE) sh -c "cd /go/src/$(PACKAGE)/server/" && gofmt -l -w $(SOURCES)


.PHONY: gen

gen:
	cd proto && \
	protoc -I chatbothub/ chatbothub/chatbothub.proto --go_out=plugins=grpc:chatbothub

