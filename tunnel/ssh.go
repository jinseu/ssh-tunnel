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
	Direct *Direct
	sf     Group
	l      sync.RWMutex
}

func initPublicKey(client *SSH){
	id_rsa := c.PrivateKey
	pem, err := ioutil.ReadFile(id_rsa)
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

func NewSSH(c *conf.Config) (client *SSH, err error) {
	client = &SSH{
		Config: c,
		CliCfg: &ssh.ClientConfig{},
	}
	client.URL, err = url.Parse(c.RemoteAddress)
	if err != nil {
		return
	}

	if client.URL.User != nil {
		client.CliCfg.User = client.URL.User.Username()
	} else {
		u, err := user.Current()
		if err != nil {
			return _, err
		}
		client.CliCfg.User = u.Username
	}

	initPublicKey(client)
	if len(client.CliCfg.Auth) == 0 {
		initPassword(client)
	}

	if len(self.CliCfg.Auth) == 0 {
		err = errors.New("Invalid auth method, please add password or generate ssh keys")
		return
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

	client.Direct = &Direct{
		Tr: &http.Transport{Dial: dial},
	}
	return
}

func (client *SSH) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	client.Direct.ServeHTTP(w, r)
}

func (client *SSH) Connect(w http.ResponseWriter, r *http.Request) {
	client.Direct.Connect(w, r)
}