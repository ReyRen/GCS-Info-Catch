package main

import (
	"github.com/sevlyar/go-daemon"
	"log"
	"net"
	"net/rpc"
)

// HelloService is rpc server obj
type GCSInfoCatchService struct{}

// Hello is rpc server method x.
func (g *GCSInfoCatchService) Hello(request string, reply *string) error {
	*reply = "hello:" + request
	return nil
}

// GoodLuck is rpc server method x
func (g *GCSInfoCatchService) GoodLuck(request string, reply *string) error {
	*reply = "Good_luck:" + request
	return nil
}

func main() {
	//Setup daemon system
	cntxt := &daemon.Context{
		PidFileName: "gcsInfoCatch.pid",
		PidFilePerm: 0644,
		LogFileName: "./log/gcsInfoCatch.log",
		WorkDir:     "./",
		Umask:       027,
		Args:        []string{"[gcsInfoCatch]"},
	}
	if len(daemon.ActiveFlags()) > 0 {
		d, err := cntxt.Search()
		if err != nil {
			log.Fatal("cntxt.Search error:", err.Error())
		}
		daemon.SendCommands(d)
		return
	}
	d, err := cntxt.Reborn()
	if err != nil {
		log.Fatal("cntxt.Search error:", err.Error())
	}
	if d != nil {
		return
	}
	defer cntxt.Release()
	log.Println("- - - - - - -[GCS-Info-Catch] started - - - - - - -")
	defer func() {
		log.Println("- - - - - - -[GCS-Info-Catch] exited - - - - - - -")
	}()
	//Daemon system ready

	rpc.RegisterName("gcs-info-catch-service", new(GCSInfoCatchService))
	listener, err := net.Listen("tcp", "172.18.127.62:40062")
	if err != nil {
		log.Fatal("ListenTCP error:", err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("Accept error:", err)
		}
		go func() {
			rpc.ServeConn(conn)
		}()
	}
}
