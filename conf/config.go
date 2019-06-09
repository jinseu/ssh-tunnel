package conf

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sort"
)

type Config struct {
	PrivateKey    string   `json:"private_key"`
	LocalAddress  string   `json:"local_address"`
	RemoteAddress string   `json:"remote_address"`
	ProxyTimeout  int      `json:"proxy_timeout"`
	BlockedList   []string `json:"blocked"`
}

// Load file from path
func NewConfig(path string) (*Config, error) {
	conf := &Config{}
	path = os.ExpandEnv(path)
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(buf, conf)
	if err != nil {
		return nil, err
	}
	conf.PrivateKey = os.ExpandEnv(conf.PrivateKey)
	sort.Strings(conf.BlockedList)
	return conf, nil
}

// test whether host is in blocked list or not
func (conf *Config) IsBlocked(host string) bool {
	i := sort.SearchStrings(conf.BlockedList, host)
	return i < len(conf.BlockedList) && conf.BlockedList[i] == host
}