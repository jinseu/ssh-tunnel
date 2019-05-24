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
		logger.Error(err)
	}

	logger.Printf("Connecting remote SSH server: %s\n", c.RemoteServer)

	wait := make(chan int)

	go func() {
		smart, err := tunnel.NewServer(c)
		if err != nil {
			L.Fatalln(err)
		}
		logger.Info("Local smart HTTP proxy: %s\n", c.File.LocalSmartServer)
		logger.Error(http.ListenAndServe(c.LocalAddress, smart))
		wait <- 1
	}()
	<-wait
}

func getPublicSuffix(domain string) {
	tld, _ := publicsuffix.EffectiveTLDPlusOne(domain)
	fmt.Printf("EffectiveTLDPlusOne: %s\n", tld)
	suffix, _ := publicsuffix.PublicSuffix(domain)
	fmt.Printf("PublicSuffix: %s\n", suffix)
}

func reload(configFile string) {
	file, err := NewConfigFile(os.ExpandEnv(configFile))
	if err != nil {
		L.Fatal(err)
	}
	res, err := http.Get(fmt.Sprintf("http://%s/reload", file.LocalNormalServer))
	if err != nil {
		L.Fatal(err)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		L.Fatal(err)
	}
	fmt.Printf("%s\n", body)
}

func main() {
	configFile   := flag.String("config", "$HOME/.config/mallory.json", "config file")
	publicSuffix := flag.String("suffix", "", "print pulbic suffix for the given domain")
	reload       := flag.Bool("reload", false, "send signal to reload config file")
	logDir       := flag.String("log-dir", "./", "log dir")

	flag.Parse()

	logger.InitGlobalLogger(*logDir, "client", 1024 * 1024 * 100, logger.INFO)

	if *FSuffix != "" {
		getPublicSuffix(*publicSuffix)
	} else if *FReload {
		reload(*configFile)
	} else {
		serve(*configFile)
	}
}
