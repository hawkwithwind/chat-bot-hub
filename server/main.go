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
	Streaming streaming.Config
}

var (
	configPath = flag.String("c", "config/config.yml", "config file path")
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

	data := make([]byte, 2048)
	len := 0
	for {
		n, _ := config.Read(data)
		if 0 == n {
			break
		}
		len += n
	}
	yaml.Unmarshal(data[:len], &c)

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

	c.Hub.Database = c.Web.Database
	c.Hub.Mongo = c.Web.Mongo

	c.Streaming.Database = c.Web.Database
	c.Streaming.Mongo = c.Web.Mongo
	c.Streaming.WebBaseUrl = c.Web.Baseurl

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
		go func() {
			wg.Add(1)
			defer wg.Done()
			config.Web.Redis = config.Redis
			config.Web.Fluent = config.Fluent
			webserver := web.WebServer{
				Config:  config.Web,
				Hubhost: "127.0.0.1",
				Hubport: config.Hub.Port}
			webserver.Serve()
		}()
	}

	if *startcmd == "hub" {
		go func() {
			wg.Add(1)
			defer wg.Done()

			raven.CapturePanicAndWait(func() {
				config.Hub.Redis = config.Redis
				config.Hub.Fluent = config.Fluent
				hub := chatbothub.ChatHub{
					Config:     config.Hub,
					Webhost:    "web",
					Webport:    config.Web.Port,
					WebBaseUrl: config.Web.Baseurl,
				}
				hub.Serve()
			}, nil)
		}()
	}

	if *startcmd == "tasks" {
		go func() {
			wg.Add(1)
			//defer wg.Done()

			task := tasks.Tasks{
				Webhost:    "web",
				Webport:    config.Web.Port,
				WebBaseUrl: config.Web.Baseurl,
			}

			err := task.Serve()
			if err != nil {
				wg.Done()
				log.Printf("task start failed %s\n", err)
			}
		}()
	}

	if *startcmd == "streaming" {
		go func() {
			wg.Add(1)
			defer wg.Done()

			server := streaming.Server{
				Config: config.Streaming,
			}

			err := server.Serve()
			if err != nil {
				wg.Done()
				log.Printf("streaming start failed %s\n", err.Error())
			}
		}()
	}

	time.Sleep(5 * time.Second)
	wg.Wait()
	log.Printf("server ends.")
}
