package tunnel

import (
	"errors"
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

func initPublicKey(client *SSH){
	pem, err := ioutil.ReadFile(c.PrivateKey)
	if err != nil {
		logger.Error("ReadFile %s failed:%s\n", c.PrivateKey, err)
		return
	}
	signer, err := ssh.ParsePrivateKey(pem)
	if err != nil {
		logger.Error("ParsePrivateKey %s failed:%s\n", c.PrivateKey, err)
		return
	}
	ssh.CliCfg.Auth = append(ssh.CliCfg.Auth, ssh.PublicKeys(signer))
}

func initPassword(client *SSH){
	if pass, ok := client.URL.User.Password(); ok {
		client.CliCfg.Auth = append(client.CliCfg.Auth, ssh.Password(pass))
	}

}

func NewSSH(c *conf.Config) *http.Transport {
	client := &SSH{
		Config: c,
		CliCfg: &ssh.ClientConfig{},
	}
	client.URL, err = url.Parse(c.RemoteAddress)
	if err != nil {
		return nil
	}

	if client.URL.User != nil {
		client.CliCfg.User = client.URL.User.Username()
	} else {
		u, err := user.Current()
		if err != nil {
			logger.Info("GET Current User Error%s", err.Error())
			return nil
		}
		client.CliCfg.User = u.Username
	}

	initPublicKey(client)
	if len(client.CliCfg.Auth) == 0 {
		initPassword(client)
	}

	if len(self.CliCfg.Auth) == 0 {
		logger.Fa("Invalid auth method, please add password or generate ssh keys")
		return nil
	}

	client.Client, err = ssh.Dial("tcp", self.URL.Host, self.CliCfg)
	if err != nil {
		return
	}

	dial := func(network, addr string) (c net.Conn, err error) {
		self.l.RLock()
		cli := self.Client
		self.l.RUnlock()

		c, err = cli.Dial(network, addr)
		if err == nil {
			return
		}

		L.Printf("dial %s failed: %s, reconnecting ssh server %s...\n", addr, err, self.URL.Host)

		clif, err := self.sf.Do(network+addr, func() (interface{}, error) {
			return ssh.Dial("tcp", self.URL.Host, self.CliCfg)
		})
		if err != nil {
			L.Printf("connect ssh server %s failed: %s\n", self.URL.Host, err)
			return
		}
		cli = clif.(*ssh.Client)

		self.l.Lock()
		self.Client = cli
		self.l.Unlock()

		return cli.Dial(network, addr)
	}

	return &http.Transport{Dial: dial}
}

func (client *SSH) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	client.Direct.ServeHTTP(w, r)
}

func (client *SSH) Connect(w http.ResponseWriter, r *http.Request) {
	client.Direct.Connect(w, r)
}
