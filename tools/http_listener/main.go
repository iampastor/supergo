package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
	// "github.com/prometheus/client_golang/prometheus/promhttp"
)

var l net.Listener
var err error
var wg sync.WaitGroup

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("start http server")
	fileListener := os.NewFile(3, "ghttpserver")
	if fileListener == nil {
		log.Panic("invalid fd")
	} else {
		l, err = net.FileListener(fileListener)
		if err != nil {
			log.Panic(err)
		}
		fileListener.Close()
		httpServer := &http.Server{}
		// http.Handle("/metrics", promhttp.Handler())
		http.HandleFunc("/hello", func(resp http.ResponseWriter, req *http.Request) {
			name := req.FormValue("name")
			if name == "" {
				name = "Jack"
			}
			resp.Write([]byte(fmt.Sprintf("hello, %s", name)))
		})
		/* http.HandleFunc("/world", func(resp http.ResponseWriter, req *http.Request) {
			resp.Write([]byte("hello, world"))
		}) */
		http.HandleFunc("/sleep", func(resp http.ResponseWriter, req *http.Request) {
			wg.Add(1)
			defer wg.Done()
			var sleepTime time.Duration
			t := req.FormValue("time")
			if t == "" {
				sleepTime = time.Second * 60
			} else {
				ti, err := strconv.Atoi(t)
				if err != nil {
					resp.Write([]byte("invalid sleep time"))
					return
				}
				sleepTime = time.Duration(ti) * time.Second
			}
			log.Printf("sleep %v", sleepTime)
			time.Sleep(sleepTime)

		})
		http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
			pid := os.Getpid()
			resp.Write([]byte(fmt.Sprintf("pid: %d", pid)))
		})
		go func() {
			err = httpServer.Serve(l)
			if err != nil {
				log.Printf("metrics server: %s", err.Error())
			}
		}()
	}
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("got signal: %+v", sig)
	switch sig {
	case syscall.SIGTERM:
		l.Close()
		wg.Wait()
		log.Println("stop http server")
		os.Exit(0)
	}
}
