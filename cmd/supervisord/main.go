package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/iampastor/supervisor/supervisord"
)

var (
	configFile string
	listenAddr string
)

func init() {
	v := flag.Bool("version", false, "print version info & exit")
	flag.StringVar(&configFile, "config", "config/supervisord.toml", "supervisord config file path")
	flag.StringVar(&listenAddr, "listen", "127.0.0.1:22106", "listen address")
	flag.Parse()

	if *v {
		PrintVersion()
		os.Exit(0)
	}
}

func main() {
	cfg, err := supervisord.ParseConfigFile(configFile)
	if err != nil {
		log.Panic(err)
	}
	super := supervisord.NewSupervisor(cfg)
	for name, p := range cfg.ProgramConfigs {
		p, err := super.AddProgram(name, p)
		if err != nil {
			log.Panic(err)
		}
		p.StartProcess()
	}

	l, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Panic(err)
	}
	go super.ServeHTTP(l)

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, syscall.SIGSTOP)
	for {
		s := <-c
		log.Printf("get a signal %s", s.String())
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGSTOP, syscall.SIGINT:
			super.Exit()
			return
		case syscall.SIGHUP:
		default:
			return
		}
	}
}
