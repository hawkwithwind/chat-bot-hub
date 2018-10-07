package main

import (
	"log"
	"sync"
	"time"
	"flag"
	"os"
	
	"gopkg.in/yaml.v2"
)

type RedisConfig struct {
	Host string
	Port string
	Db string
}

type WebConfig struct {
	Host string
	Port string
	User string
	Pass string
	SecretPhrase string
	Redis RedisConfig
}

var (
	webConfigPath = flag.String("c", "config/config.yml", "config file path")
)

func loadWebConfig(configPath string) (WebConfig, error) {
	c := WebConfig{}
	
	config, err := os.Open(configPath)
	defer config.Close()
	if err != nil {
		return c, err
	}

	data := make([]byte, 1024)
	len := 0
	for {
		n, _ := config.Read(data)
		if 0 == n { break }
		len += n
	}
	yaml.Unmarshal(data[:len], &c)
	return c, nil
}

func main() {
	flag.Parse()
	log.SetPrefix("[MAIN]")
	log.Printf("config path %s", *webConfigPath)
	
	var wg sync.WaitGroup
	log.Printf("server starts.")
		
	go func() {
		wg.Add(1)
		defer wg.Done()

		if webconfig, err := loadWebConfig(*webConfigPath); err != nil {
			log.Fatalf("failed to open config file %s, exit.", err)
			return
		} else {
			webserver := WebServer{config: webconfig}
			webserver.serve()
		}
	}()
	
	go func() {
		wg.Add(1)
		defer wg.Done()

		qqhub := QQHub{}
		qqhub.serve()		
	}()

	time.Sleep(5 * time.Second)
	wg.Wait()
	log.Printf("server ends.")
}
