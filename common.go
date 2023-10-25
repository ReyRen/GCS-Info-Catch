package main

import "strings"

var (
	//本身启动的 RPC server 监听地址（本机地址）
	RPC_ADDDR_AND_PORT = "172.18.127.62:50001"

	//私有镜像仓库的配置信息
	DOCKER_REGISTRY_USERNAME = "admin"
	DOCKER_REGISTRY_PASSWORD = "adminADMIN123"
	DOCKER_REGISTRY_ADDRESS  = "http://172.18.127.68:80"

	//创建的 docker swarm 的 overlay 网络名字
	MY_OVERLAY_NETWORK = "my-attachable-overlay"

	//映射存储
	SOURCE_ALL      = "/storage-ftp-data"
	TARGET_ALL      = "/storage-root"
	SOURCE_DATASETS = "/storage-ftp-data/datasets"
	TARGET_DATASETS = "/storage-root/datasets"

	//entrypoint
	startScript = "/storage-root/script/start.py"
)

// GCSInfoCatchServer is rpc server obj
type GCSInfoCatchServer struct{}

func stringTrimHandler(srcString string) []string {
	return strings.Split(srcString, ",")
}
