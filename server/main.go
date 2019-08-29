package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/getsentry/raven-go"
	"gopkg.in/yaml.v2"

	"github.com/hawkwithwind/chat-bot-hub/server/chatbothub"
	"github.com/hawkwithwind/chat-bot-hub/server/streaming"
	"github.com/hawkwithwind/chat-bot-hub/server/tasks"
	"github.com/hawkwithwind/chat-bot-hub/server/utils"
	"github.com/hawkwithwind/chat-bot-hub/server/web"
)

type MainConfig struct {
	Hub       chatbothub.ChatHubConfig
	Web       web.WebConfig
	Redis     utils.RedisConfig
	Fluent    utils.FluentConfig
	Rabbitmq  utils.RabbitMQConfig
	Streaming streaming.Config
}

var (
	configPath = flag.String("c", "/config/config.yml", "config file path")
	startcmd   = flag.String("s", "", "start command: web/hub")
	config     MainConfig
)

func loadConfig(configPath string) (MainConfig, error) {
	c := MainConfig{}

	config, err := os.Open(configPath)
	defer config.Close()
	if err != nil {
		return c, err
	}

	data := make([]byte, 16*1024)
	len := 0
	for {
		n, _ := config.Read(data)
		if 0 == n {
			break
		}
		len += n
	}

	err = yaml.Unmarshal(data[:len], &c)
	if err != nil {
		return c, err
	}

	dbuser := os.Getenv("DB_USER")
	dbpassword := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	dblink := os.Getenv("DB_ALIAS")
	dbparams := os.Getenv("DB_PARAMS")
	dbmaxconn := os.Getenv("DB_MAXCONN")

	c.Web.Database.DataSourceName = fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", dbuser, dbpassword, dblink, dbname, dbparams)

	if dbmaxconn != "" {
		maxconn, err := strconv.Atoi(dbmaxconn)
		if err == nil {
			c.Web.Database.MaxConnectNum = maxconn
		}
	}

	c.Hub.Mongo = c.Web.Mongo

	c.Streaming.Mongo = c.Web.Mongo
	c.Streaming.WebBaseUrl = c.Web.Baseurl
	c.Streaming.Oss = c.Hub.Oss
	c.Web.Oss = c.Hub.Oss

	return c, nil
}

func main() {
	flag.Parse()
	log.SetPrefix("[MAIN]")
	log.Printf("config path %s", *configPath)

	var wg sync.WaitGroup
	log.Printf("server %s starts.", *startcmd)

	var err error
	if config, err = loadConfig(*configPath); err != nil {
		log.Fatalf("failed to open config file %s, exit.", err)
		return
	}

	raven.SetDSN(config.Web.Sentry)

	if *startcmd == "web" {
		wg.Add(1)

		go func() {
			defer wg.Done()
			config.Web.Redis = config.Redis
			config.Web.Fluent = config.Fluent
			config.Web.Rabbitmq = config.Rabbitmq

			webserver := web.WebServer{
				Config:  config.Web,
				Hubhost: "hub",
				Hubport: config.Hub.Port}
			webserver.Serve()
		}()
	}

	if *startcmd == "hub" {
		wg.Add(1)

		go func() {
			defer wg.Done()

			raven.CapturePanicAndWait(func() {
				config.Hub.Redis = config.Redis
				config.Hub.Fluent = config.Fluent
				config.Hub.Rabbitmq = config.Rabbitmq

				hub := chatbothub.ChatHub{
					Config:          config.Hub,
					Webhost:         "web",
					Webport:         config.Web.Port,
					WebBaseUrl:      config.Web.Baseurl,
					WebSecretPhrase: config.Web.SecretPhrase,
				}
				hub.Serve()
			}, nil)
		}()
	}

	if *startcmd == "tasks" {
		wg.Add(1)

		go func() {
			//defer wg.Done()

			task := tasks.Tasks{
				Webhost:    "web",
				Webport:    config.Web.Port,
				WebBaseUrl: config.Web.Baseurl,
			}

			err := task.Serve()
			if err != nil {
				log.Printf("task start failed %s\n", err)
			}
		}()
	}

	if *startcmd == "streaming" {
		wg.Add(1)

		go func() {
			defer wg.Done()

			server := streaming.Server{
				Config: config.Streaming,
			}

			err := server.Serve()
			if err != nil {
				log.Printf("streaming start failed %s\n", err.Error())
			}
		}()
	}

	time.Sleep(5 * time.Second)
	wg.Wait()
	log.Printf("server ends.")
}
