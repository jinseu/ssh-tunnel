package tunnel

import (
	"net/http"
	"sync"
	//"time"

	"golang.org/x/net/publicsuffix"

	"jinseu/ssh-tunnel/conf"
	"jinseu/ssh-tunnel/logger"
	"jinseu/ssh-tunnel/util"
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
	BlockedHosts map[string]bool
	mutex sync.RWMutex
}

// Create and intialize
func NewServer(c *conf.Config) (svr *Server, err error) {
    prtc := NewProtocol(c)
	svr = &Server{
		Cfg: c,
		Prtc: prtc,
		BlockedHosts: make(map[string]bool),
	}
	return
}

func (s *Server) Blocked(host string) bool {
	blocked, cached := false, false
	host = util.GetHost(host)
	s.mutex.RLock()
	if s.BlockedHosts[host] {
		blocked = true
		cached = true
	}
	s.mutex.RUnlock()

	if !blocked {
		tld, _ := publicsuffix.EffectiveTLDPlusOne(host)
		blocked = s.Cfg.IsBlocked(tld)
	}

	if !blocked {
		suffix, _ := publicsuffix.PublicSuffix(host)
		blocked = s.Cfg.IsBlocked(suffix)
	}

	if blocked && !cached {
		s.mutex.Lock()
		s.BlockedHosts[host] = true
		s.mutex.Unlock()
	}
	return blocked
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "CONNECT" {
		s.Prtc.Connect(w, r)
	} else if r.URL.IsAbs() {
		r.RequestURI = ""
		util.RemoveHopHeaders(r.Header)
		s.Prtc.ServeHTTP(w, r)
	} else {
		logger.Error("%s is not a full URL path\n", r.RequestURI)
	}
}
