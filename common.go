package main

import (
	"errors"
	"fmt"

	//"net"
	"net"
	"strings"
)

var (
	//本身启动的 RPC server 监听地址（本机地址）
	//RPC_ADDDR_AND_PORT = "172.18.127.101:50001"
	RPC_INTERFACE_NAME = "ens11f0"

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
	startScript = "/storage-root/script/start1.sh"
)

// GCSInfoCatchServer is rpc server obj
type GCSInfoCatchServer struct{}

func getLocalIpAddress(interfaceName string) (addr string, err error) {
	//ens11f0
	var (
		ief      *net.Interface
		addrs    []net.Addr
		ipv4Addr net.IP
	)
	if ief, err = net.InterfaceByName(interfaceName); err != nil { // get interface
		return
	}
	if addrs, err = ief.Addrs(); err != nil { // get addresses
		return
	}
	for _, addr := range addrs { // get ipv4 address
		if ipv4Addr = addr.(*net.IPNet).IP.To4(); ipv4Addr != nil {
			break
		}
	}
	if ipv4Addr == nil {
		return "", errors.New(fmt.Sprintf("interface %s don't have an ipv4 address\n", interfaceName))
	}
	return ipv4Addr.String(), nil
}

func stringTrimHandler(srcString string) []string {
	return strings.Split(srcString, ",")
}
