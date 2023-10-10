package main

import (
	"log"
	"net"
	"net/rpc"
)

/*
// Hello is rpc server method x.
func (g *GCSInfoCatchService) getContainerInfo(request string, reply *string) error {
	*reply = "hello:" + request
	return nil
}

// GoodLuck is rpc server method x
func (g *GCSInfoCatchService) GoodLuck(request string, reply *string) error {
	*reply = "Good_luck:" + request
	return nil
}
*/

func run_handler() {
	err := rpc.RegisterName(RPC_REGISTER_NAME, new(GCSInfoCatchService))
	if err != nil {
		log.Fatal("rpc.RegisterName error:", err.Error())
	}
	log.Println("rpc.RegisterName [gcs-info-catch-service] done")
	listener, err := net.Listen("tcp", RPC_ADDDR_AND_PORT)
	if err != nil {
		log.Fatal("ListenTCP error:", err.Error())
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("Accept error:", err.Error())
		}
		go func() {
			rpc.ServeConn(conn)
		}()
	}
}
