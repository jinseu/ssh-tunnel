package tunnel

import (
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"jinseu/ssh-tunnel/conf"
	"jinseu/ssh-tunnel/logger"
)

var (
	ErrShouldProxy = errors.New("should proxy")
)

type Protocol struct {
	SSHTransport *http.Transport
	HTTPTransport *http.Transport
}

func NewProtocol(c *conf.Config) *Protocol {
	shouldProxyTimeout := 200 * time.Millisecond
	transport := http.DefaultTransport.(*http.Transport)
	transport.Dial = (&net.Dialer{
		Timeout: shouldProxyTimeout,
	}).Dial

	sshTransport, _ := NewSSH(c)
	return &Protocol{
		SSHTransport: sshTransport,
		HTTPTransport: transport,
	}
}

func (prtc *Protocol) ServeHTTP(w http.ResponseWriter, r *http.Request) (err error) {
	if r.Method == "CONNECT" {
		logger.Error("this function can not handle CONNECT method")
		http.Error(w, r.Method, http.StatusMethodNotAllowed)
		return
	}
	start := time.Now()

	resp, err := prtc.Tr.RoundTrip(r)
	if err != nil {
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			L.Printf("RoundTrip: %s, reproxy...\n", err.Error())
			err = ErrShouldProxy
			return
		}
		L.Printf("RoundTrip: %s\n", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// please prepare header first and write them
	CopyHeader(w, resp)
	w.WriteHeader(resp.StatusCode)

	n, err := io.Copy(w, resp.Body)
	if err != nil {
		L.Printf("Copy: %s\n", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	d := FormatHumDuration(time.Since(start))
	ndtos := FormatHunSize(n)
	L.Printf("RESPONSE %s %s in %s <-%s\n", r.URL.Host, resp.Status, d, ndtos)
	return
}

func (prtc *Protocol) Connect(w http.ResponseWriter, r *http.Request) (err error) {
	if r.Method != "CONNECT" {
		L.Println("this function can only handle CONNECT method")
		http.Error(w, r.Method, http.StatusMethodNotAllowed)
		return
	}
	start := time.Now()

	// Use Hijacker to get the underlying connection
	hij, ok := w.(http.Hijacker)
	if !ok {
		s := "Server does not support Hijacker"
		logger.Error(s)
		http.Error(w, s, http.StatusInternalServerError)
		return
	}

	// connect the remote client directly
	dst, err := prtc.Tr.Dial("tcp", r.URL.Host)
	if err != nil {
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			logger.Error("RoundTrip: %s, reproxy...\n", err.Error())
			err = ErrShouldProxy
			return
		}
		logger.Error("Dial: %s\n", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	src, _, err := hij.Hijack()
	if err != nil {
		logger.Error("Hijack: %s\n", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer src.Close()

	src.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))

	copyAndWait := func(dst io.Writer, src io.Reader, c chan int64) {
		n, err := io.Copy(dst, src)
		if err != nil {
			logger.Error("Copy: %s\n", err.Error())
			// FIXME: how to report error to dst ?
		}
		c <- n
	}

	// client to remote
	stod := make(chan int64)
	go copyAndWait(dst, src, stod)

	dtos := make(chan int64)
	go copyAndWait(src, dst, dtos)

	nstod, ndtos := FormatHunSize(<-stod), FormatHunSize(<-dtos)
	d := FormatHumDuration(time.Since(start))
	L.Printf("CLOSE %s after %s ->%s <-%s\n", r.URL.Host, d, nstod, ndtos)
	return
}
