package main

import (
	pb "GCS-Info-Catch/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"io"
	"log"
	"net"
	"time"
)

func run_handler_server() {

	lis, err := net.Listen("tcp", RPC_ADDDR_AND_PORT)
	if err != nil {
		log.Printf("Failed to listen: %v\n", err.Error())
	}

	// 实例化grpc服务端
	s := grpc.NewServer()

	// 注册Greeter服务
	pb.RegisterGcsInfoCatchServiceServer(s, &GCSInfoCatchServer{})

	// 往grpc服务端注册反射服务
	reflection.Register(s)
	log.Println("reflection.Register ok")

	// 启动grpc服务
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func run_handler_client() {
	// 连接grpc服务器
	conn, err := grpc.Dial(RPC_ADDDR_AND_PORT, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("did not connect: %v\n", err)
	}
	// 延迟关闭连接
	defer conn.Close()

	// 初始化Greeter服务客户端
	c := pb.NewGreeterClient(conn)

	// 初始化上下文，设置请求超时时间为1秒
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// 延迟关闭请求会话
	defer cancel()

	// 2.调用获取stream
	stream, err := c.SayHello(ctx, &pb.HelloRequest{
		Name: "fuck the world",
		Age:  30,
	})
	if err != nil {
		log.Fatalf("could not echo: %v", err)
	}

	// 3. for循环获取服务端推送的消息
	for {
		// 通过 Recv() 不断获取服务端send()推送的消息
		resp, err := stream.Recv()
		// 4. err==io.EOF则表示服务端关闭stream了 退出
		if err == io.EOF {
			log.Println("server closed")
			break
		}
		if err != nil {
			log.Printf("Recv error:%v", err)
			continue
		}
		log.Printf("Recv data:%v", resp.GetMessage())
	}
}
