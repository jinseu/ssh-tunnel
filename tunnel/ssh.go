package tunnel

import (
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os/user"
	"sync"

	"golang.org/x/crypto/ssh"

	"jinseu/ssh-tunnel/conf"
	"jinseu/ssh-tunnel/logger"
)

type SSH struct {
	Config *conf.Config
	URL    *url.URL
	Client *ssh.Client
	CliCfg *ssh.ClientConfig
	l      sync.RWMutex
}

func initPublicKey(conf *conf.Config, cliCfg *ssh.ClientConfig) error {
	pem, err := ioutil.ReadFile(conf.PrivateKey)
	if err != nil {
		logger.Error("ReadFile %s failed:%s\n", conf.PrivateKey, err)
		return err
	}
	signer, err := ssh.ParsePrivateKey(pem)
	if err != nil {
		logger.Error("ParsePrivateKey %s failed:%s\n", conf.PrivateKey, err)
		return err
	}
	cliCfg.Auth = append(cliCfg.Auth, ssh.PublicKeys(signer))
	return nil
}

func initPassword(cliCfg *ssh.ClientConfig, URL *url.URL){
	if pass, ok := URL.User.Password(); ok {
		cliCfg.Auth = append(cliCfg.Auth, ssh.Password(pass))
	}
}

func NewSSH(c *conf.Config) *http.Transport {
	cliCfg := &ssh.ClientConfig{}

	URL, err := url.Parse(c.RemoteAddress)
	if err != nil {
		return nil
	}

	if URL.User != nil {
		cliCfg.User = URL.User.Username()
	} else {
		u, err := user.Current()
		if err != nil {
			logger.Info("GET Current User Error%s", err.Error())
			return nil
		}
		cliCfg.User = u.Username
	}

	initPublicKey(c, cliCfg)
	if len(cliCfg.Auth) == 0 {
		initPassword(cliCfg, URL)
	}

	if len(cliCfg.Auth) == 0 {
		logger.Fatal("Invalid auth method, please add password or generate ssh keys")
		return nil
	}

	client, err := ssh.Dial("tcp", URL.Host, cliCfg)
	if err != nil {
		logger.Error(err.Error())
		return nil
	}

	dial := func(network, addr string) (c net.Conn, err error) {

		c, err = client.Dial(network, addr)
		if err != nil {
			logger.Info("dial %s failed: %s, ", addr, err)
			return
		}
        return c, nil
	}

	return &http.Transport{Dial: dial}
}

