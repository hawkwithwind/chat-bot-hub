# chat-bot-hub

chat bot hub build upon gRPC

# How to build and run

```
#/ make
#/ docker-compose up -d
```
please reference "Build environment requirement" section below for more details.

# What does it do

it will starts a web server and a grpc server, grpc server will receive chat-bot register message

web server can control and get info from grpc server by rpc call. in this way, we can provide a user interface to control the bots.

```
  +-------------+
  | web server  |
  +-------------+
       |  ^
       v  | rpc call (for s[web])
  +------------------------+
  | grpc server (chat hub) |
  +------------------------+
       |  ^
       v  | rpc call (for c[ChatBot])
  +-------------+
  | bot client  |  .... (currently have c[QQBOT] and c[WECHATBOT])
  +-------------+
       | ^
       v | pcqq-protocol, padchat-protocol
  +-----------------+
  | tencent server  | ... (to control the real qq or wechat accounts)
  +-----------------+

```

# Build environment requirement

## Docker deamon

this project uses docker to complie/pack and run all the code, so you need docker deamon installed and running on your local machine.

## GOPATH

if you have golang installed on your local machine, better to use ```go get```

```
#/ go get github.com/hawkwithwind/chat-bot-hub
```

this project uses docker to complie and run all the code, so that if you don't have golang installed, you can still complie and run the project. but the makefile assumes `$GOPATH` directroy

```
#/ export GOPATH=~/golang_workspace
#/ mkdir -p $GOPATH/src/github.com/hawkwithwind
#/ cd $GOPATH/src/github.com/hawkwithwind
#/ git clone git@github.com/hawkwithwind/chat-bot-hub
#/ cd chat-bot-hub
#/ make
```

## Internet access

docker image build, golang and nodejs build are all need internet access. makefile will make use of ```http_proxy``` and ```https_proxy``` environment and send those to docker container while compiling and packing golang and nodejs code.

for the same reason, the Dockerfiles ```docker/*/Dockerfile``` uses China mirrors for alpine and nodjes registory. if you are in different location, feel free to comment out those lines, or change to other mirrors.


- docker/runtime/Dockerfile

```
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
```

- docker/build/golang/Dockerfile

```
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
```

- docker/build/nodejs/Dockerfile

```
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories

 ...
 
RUN npm config set registry=http://registry.npm.taobao.org
```

