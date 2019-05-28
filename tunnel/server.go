package tunnel

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/publicsuffix"

	"jinseu/ssh-tunnel/conf"
	"jinseu/ssh-tunnel/logger"
)

const (
	SmartSrv = iota
	NormalSrv
)

type AccessType bool

func (t AccessType) String() string {
	if t {
		return "PROXY"
	} else {
		return "DIRECT"
	}
}

type Server struct {
	Cfg    *conf.Config
	Prtc   *Protocol
	SSH    *SSH
	BlockedHosts map[string]bool
	mutex sync.RWMutex
}

// Create and intialize
func NewServer(c *conf.Config) (svr *Server, err error) {
	ssh := NewSSH(c)
	if err != nil {
		return
	}

	shouldProxyTimeout := time.Millisecond * time.Duration(c.File.ShouldProxyTimeoutMS)

	svr = &Server{
		Cfg:          c,
		Direct:       NewDirect(shouldProxyTimeout),
		SSH:          ssh,
		BlockedHosts: make(map[string]bool),
	}
	return
}

func (self *Server) Blocked(host string) bool {
	blocked, cached := false, false
	host = GetHost(host)
	self.mutex.RLock()
	if self.BlockedHosts[host] {
		blocked = true
		cached = true
	}
	self.mutex.RUnlock()

	if !blocked {
		tld, _ := publicsuffix.EffectiveTLDPlusOne(host)
		blocked = self.Cfg.Blocked(tld)
	}

	if !blocked {
		suffix, _ := publicsuffix.PublicSuffix(host)
		blocked = self.Cfg.Blocked(suffix)
	}

	if blocked && !cached {
		self.mutex.Lock()
		self.BlockedHosts[host] = true
		self.mutex.Unlock()
	}
	return blocked
}

func (svr *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	use := (svr.Blocked(r.URL.Host) || svr.Mode == NormalSrv) && r.URL.Host != ""
	logger.Info("[%s] %s %s %s\n", AccessType(use), r.Method, r.RequestURI, r.Proto)

	if r.Method == "CONNECT" {
		if use {
			svr.SSH.Connect(w, r)
		} else {
			err := svr.Direct.Connect(w, r)
			if err == ErrShouldProxy {
				svr.SSH.Connect(w, r)
			}
		}
	} else if r.URL.IsAbs() {
		r.RequestURI = ""
		RemoveHopHeaders(r.Header)
		if use {
			svr.SSH.ServeHTTP(w, r)
		} else {
			err := svr.Direct.ServeHTTP(w, r)
			if err == ErrShouldProxy {
				svr.SSH.ServeHTTP(w, r)
			}
		}
	} else if r.URL.Path == "/reload" {
		svr.reload(w, r)
	} else {
		logger.Error("%s is not a full URL path\n", r.RequestURI)
	}
}

func (svr *Server) reload(w http.ResponseWriter, r *http.Request) {
	err := svr.Cfg.Reload()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(self.Cfg.Path + ": " + err.Error()))
	} else {
		w.WriteHeader(200)
		w.Write([]byte(self.Cfg.Path + " reloaded"))
	}
}
