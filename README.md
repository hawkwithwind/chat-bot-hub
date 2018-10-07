# chat-bot-hub
chat bot hub build upon gRPC

# how to build and run
```
#/ make
#/ docker-compose up -d
```

# what does it do

it will starts a web server and a grpc server, grpc server will receive chat-bot register message

web server can control and get info from grpc server by rpc call. in this way, we can provide a user interface to controll the bots.

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


