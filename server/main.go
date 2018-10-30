package main

import (
	"flag"
	"log"
	"os"
	"sync"
	"time"

	"github.com/getsentry/raven-go"
	"gopkg.in/yaml.v2"
)

type MainConfig struct {
	Hub ChatHubConfig
	Web WebConfig
}

var (
	configPath = flag.String("c", "config/config.yml", "config file path")
	config     MainConfig
)

func loadConfig(configPath string) (MainConfig, error) {
	c := MainConfig{}

	config, err := os.Open(configPath)
	defer config.Close()
	if err != nil {
		return c, err
	}

	data := make([]byte, 1024)
	len := 0
	for {
		n, _ := config.Read(data)
		if 0 == n {
			break
		}
		len += n
	}
	yaml.Unmarshal(data[:len], &c)
	return c, nil
}

func main() {
	flag.Parse()
	log.SetPrefix("[MAIN]")
	log.Printf("config path %s", *configPath)

	var wg sync.WaitGroup
	log.Printf("server starts.")

	var err error
	if config, err = loadConfig(*configPath); err != nil {
		log.Fatalf("failed to open config file %s, exit.", err)
		return
	}

	raven.SetDSN(config.Web.Sentry)

	go func() {
		wg.Add(1)
		defer wg.Done()

		webserver := WebServer{config: config.Web, hubport: config.Hub.Port}
		webserver.serve()
	}()

	go func() {
		wg.Add(1)
		defer wg.Done()

		qqhub := ChatHub{config: config.Hub}
		qqhub.serve()
	}()

	time.Sleep(5 * time.Second)
	wg.Wait()
	log.Printf("server ends.")
}
