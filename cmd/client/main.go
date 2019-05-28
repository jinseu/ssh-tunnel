package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"golang.org/x/net/publicsuffix"

	"jinseu/ssh-tunnel/tunnel"
	"jinseu/ssh-tunnel/logger"
)

func serve(configFile string) {
	c, err := NewConfig(configFile)
	if err != nil {
		logger.Fatal("GET_CONFIG Error:%s", err.Error())
	}

	wait := make(chan int)

	go func() {
		smart, err := tunnel.NewServer(c)
		if err != nil {
			logger.Fatal("LOCAL_SERVER Error:%s", err.Error())
		}
		logger.Info("LOCAL_SERVER HTTP proxy: %s\n", c.File.LocalSmartServer)
		logger.Error(http.ListenAndServe(c.LocalAddress, smart))
		wait <- 1
	}()
	<- wait
}

func main() {
	configFile   := flag.String("config", "$HOME/.config/mallory.json", "config file")
	logDir       := flag.String("log-dir", "./", "log dir")

	flag.Parse()
	logger.InitGlobalLogger(*logDir, "client", 1024 * 1024 * 100, logger.INFO)
	serve(*configFile)
}
