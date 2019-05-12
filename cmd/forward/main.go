package main

import (
	"flag"
	"io"
	"log"
	"net"
	"os"
	"time"

	"jinseu/ssh-tunnel/logger"
)

func main() {
	network := flag.String("network", "tcp", "network protocol")
	listen  := flag.String("listen", ":20022", "listen on this port")
	forward := flag.String("forward", ":80", "destination address and port")
	logDir  := flag.String("log-dir", "./", "log dir")

	flag.Parse()

	logger.InitGlobalLogger(logDir, "forward", 1024 * 1024 * 100, logger.INFO)

	logger.info("Listening on %s for %s...\n", *FListen, *FNetwork)
	ln, err := net.Listen(*FNetwork, *FListen)
	if err != nil {
		logger.Error("Listen Error:%s", err.Error())
	}

	for id := 0; ; id++ {
		conn, err := ln.Accept()
		if err != nil {
			logger.Error("Accept Error %d: %s\n", id, err)
			continue
		}
		logger.Info("Accept %d: new %s\n", id, conn.RemoteAddr())

		if tcpConn := conn.(*net.TCPConn); tcpConn != nil {
			logger.Info("%d: setup keepalive for TCP connection\n", id)
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(30 * time.Second)
		}

		go func(myid int, conn net.Conn) {
			defer conn.Close()
			c, err := net.Dial(*FNetwork, *FForward)
			if err != nil {
				logger.Error("%d: %s\n", myid, err)
				return
			}
			logger.Info("%d: new %s <-> %s\n", myid, c.RemoteAddr(), conn.RemoteAddr())
			defer c.Close()
			wait := make(chan int)
			go func() {
				n, err := io.Copy(c, conn)
				if err != nil {
					logger.Error("%d: %s\n", myid, err)
				}
				logger.Info("%d: %s -> %s %d bytes\n", myid, conn.RemoteAddr(), c.RemoteAddr(), n)
				wait <- 1
			}()
			go func() {
				n, err := io.Copy(conn, c)
				if err != nil {
					logger.Error("%d: %s\n", myid, err)
				}
				logger.Info("%d: %s -> %s %d bytes\n", myid, c.RemoteAddr(), conn.RemoteAddr(), n)
				wait <- 1
			}()
			<-wait
			<-wait
			logger.Info("%d: connection closed\n", myid)
		}(id, conn)
	}
}
