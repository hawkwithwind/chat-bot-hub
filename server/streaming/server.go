package streaming

import (
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/globalsign/mgo"
	"github.com/hawkwithwind/chat-bot-hub/server/dbx"
	"github.com/hawkwithwind/chat-bot-hub/server/httpx"
	"github.com/hawkwithwind/chat-bot-hub/server/rpc"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	"github.com/hawkwithwind/logger"
	"math/rand"
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

	Chathubs              []string
	ChathubWeb            string
	ChathubWebAccessToken string

	Mongo    utils.MongoConfig
	Database utils.DatabaseConfig
	Oss      utils.OssConfig

	WebBaseUrl   string
	SecretPhrase string
}

type Server struct {
	*logger.Logger

	Config Config
	chmsg  chan *pb.EventReply

	websocketConnections *sync.Map

	mongoDb *mgo.Database
	db      *dbx.Database

	restfulclient *http.Client

	ossClient *oss.Client
	ossBucket *oss.Bucket

	grpcWrapper *rpc.GRPCWrapper
}

func (server *Server) init() error {
	server.Logger = logger.New()
	server.Logger.SetPrefix("[STREAMING]")
	_ = server.Logger.Init()

	server.chmsg = make(chan *pb.EventReply, 1000)
	server.websocketConnections = &sync.Map{}

	o := &ErrorHandler{}

	server.Debug("connecting to mongo, host:%s port:%s\n", server.Config.Mongo.Host, server.Config.Mongo.Port)
	server.mongoDb = o.NewMongoConn(server.Config.Mongo.Host, server.Config.Mongo.Port)
	if o.Err != nil {
		server.Error(o.Err, "connect to mongo failed %s", o.Err)
		return o.Err
	}

	server.db = &dbx.Database{}
	if o.Connect(server.db, "mysql", server.Config.Database.DataSourceName); o.Err != nil {
		server.Error(o.Err, "connect to database failed")
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

	i := rand.Intn(len(server.Config.Chathubs))
	server.grpcWrapper = rpc.CreateGRPCWrapper(server.Config.Chathubs[i])

	return nil
}

func (server *Server) Serve() error {
	if err := server.init(); err != nil {
		return err
	}

	server.StartHubClient()

	return server.ServeWebsocketServer()
}
