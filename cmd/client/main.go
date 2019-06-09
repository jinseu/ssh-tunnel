package main

import (
	"flag"
	"net/http"

	"jinseu/ssh-tunnel/tunnel"
	"jinseu/ssh-tunnel/logger"
	"jinseu/ssh-tunnel/conf"
)

func serve(configFile string) {
	c, err := conf.NewConfig(configFile)
	if err != nil {
		logger.Fatal("GET_CONFIG Error:%s", err.Error())
	}

	wait := make(chan int)

	go func() {
		s, err := tunnel.NewServer(c)
		if err != nil {
			logger.Fatal("LOCAL_SERVER Error:%s", err.Error())
		}
		logger.Info("LOCAL_SERVER HTTP proxy: %s\n", c.LocalAddress)
		err = http.ListenAndServe(c.LocalAddress, s)
		if err != nil {
			logger.Error(err.Error())
		}
		wait <- 1
	}()
	<- wait
}

func main() {
	configFile   := flag.String("config", "$HOME/.config/tunnel.json", "config file")
	logDir       := flag.String("log-dir", "./", "log dir")

	flag.Parse()
	logger.InitGlobalLogger(*logDir, "client", 1024 * 1024 * 100, logger.INFO)
	logger.SetLog2Stdout(true)
	serve(*configFile)
}
