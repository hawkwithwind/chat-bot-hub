package streaming

import (
	"encoding/json"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/globalsign/mgo"
	chatbotweb "github.com/hawkwithwind/chat-bot-hub/proto/web"
	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
	"github.com/hawkwithwind/chat-bot-hub/server/rpc"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	"github.com/hawkwithwind/logger"
	"net/http"
	"sync"

	pb "github.com/hawkwithwind/chat-bot-hub/proto/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/domains"
)

type ErrorHandler struct {
	domains.ErrorHandler
}

type Config struct {
	Host string
	Port string

	Chathubs    []string
	ChatWebGrpc string

	Mongo utils.MongoConfig
	Oss   utils.OssConfig

	WebBaseUrl string

	ClientId   string
	ClientType string
}

type Server struct {
	*logger.Logger

	Config Config
	chmsg  chan *pb.EventReply

	websocketConnections      *sync.Map
	botAndWsConnectionSubInfo *sync.Map

	mongoDb *mgo.Database

	restfulclient *http.Client

	ossClient *oss.Client
	ossBucket *oss.Bucket

	hubGRPCWrapper *rpc.GRPCWrapper
	webGRPCWrapper *rpc.GRPCWrapper
}

func (server *Server) init() error {
	server.Logger = logger.New()
	server.Logger.SetPrefix("[STREAMING]")
	_ = server.Logger.Init()

	server.chmsg = make(chan *pb.EventReply, 1000)
	server.websocketConnections = &sync.Map{}
	server.botAndWsConnectionSubInfo = &sync.Map{}

	o := &ErrorHandler{}

	server.Debug("connecting to mongo, host:%s port:%s\n", server.Config.Mongo.Host, server.Config.Mongo.Port)
	server.mongoDb = o.NewMongoConn(server.Config.Mongo.Host, server.Config.Mongo.Port, server.Config.Mongo.Database)
	if o.Err != nil {
		server.Error(o.Err, "connect to mongo failed %s", o.Err)
		return o.Err
	}

	if o.EnsuredMongoIndexes(server.mongoDb); o.Err != nil {
		server.Error(o.Err, "mongo ensure indexes fail")
		return o.Err
	}

	server.restfulclient = httpx.NewHttpClient()

	ossClient, err := oss.New(server.Config.Oss.Region, server.Config.Oss.Accesskeyid, server.Config.Oss.Accesskeysecret, oss.UseCname(true))
	if err != nil {
		server.Error(err, "cannot create ossClient")
		return err
	}

	ossBucket, err := ossClient.Bucket(server.Config.Oss.Bucket)
	if err != nil {
		server.Error(err, "cannot get oss bucket")
		return err
	}

	server.ossClient = ossClient
	server.ossBucket = ossBucket

	server.hubGRPCWrapper = rpc.CreateGRPCWrapper(server.Config.Chathubs[0])
	server.webGRPCWrapper = rpc.CreateGRPCWrapper(server.Config.ChatWebGrpc)

	return nil
}

func (server *Server) Serve() error {
	if err := server.init(); err != nil {
		return err
	}

	server.StartHubClient()

	return server.ServeWebsocketServer()
}

func (server *Server) getBotById(botId string) (*domains.Bot, error) {
	wrapper, err := server.NewWebGRPCWrapper()
	if err != nil {
		return nil, err
	}
	defer wrapper.Cancel()

	req := &chatbotweb.GetBotRequest{BotId: botId}
	res, err := wrapper.WebClient.GetBot(wrapper.Context, req)
	if err != nil {
		return nil, err
	}

	var bot domains.Bot
	if err = json.Unmarshal(res.Payload, &bot); err != nil {
		return nil, err
	}

	return &bot, nil
}
