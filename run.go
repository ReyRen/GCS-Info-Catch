package main

import (
	pb "GCS-Info-Catch/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
)

func run_handler_server() {

	addr, err := getLocalIpAddress(RPC_INTERFACE_NAME)
	if err != nil {
		log.Printf("Failed  getLocalIpAddress: %v\n", err.Error())
	}

	lis, err := net.Listen("tcp", addr+":50001")
	if err != nil {
		log.Printf("Failed to listen: %v\n", err.Error())
	}

	// 实例化grpc服务端
	infoServer := grpc.NewServer()

	// 注册Greeter服务
	pb.RegisterGcsInfoCatchServiceDockerServer(infoServer, &GCSInfoCatchServer{})

	// 往grpc服务端注册反射服务
	reflection.Register(infoServer)
	log.Println("reflection.Register ok")

	// 启动grpc服务
	if err := infoServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
