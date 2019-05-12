package util

import (
	"fmt"
	"net"
	"net/http"
)

func GetHost(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	} else {
		return host
	}
}

func CopyHeader(w http.ResponseWriter, r *http.Response) {
	// copy headers
	dst, src := w.Header(), r.Header
	for k, _ := range dst {
		dst.Del(k)
	}
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

func StatusText(c int) string {
	return fmt.Sprintf("%d %s", c, http.StatusText(c))
}

var HopByHopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"TE",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

func RemoveHopHeaders(h http.Header) {
	for _, k := range HopByHopHeaders {
		h.Del(k)
	}
}
