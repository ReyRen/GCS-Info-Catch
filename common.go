package main

import "strings"

var (
	RPC_ADDDR_AND_PORT = "172.18.127.62:50001"

	DOCKER_REGISTRY_USERNAME = "admin"
	DOCKER_REGISTRY_PASSWORD = "adminADMIN123"
	DOCKER_REGISTRY_ADDRESS  = "http://172.18.127.68:80"
)

// GCSInfoCatchServer is rpc server obj
type GCSInfoCatchServer struct{}

func stringTrimHandler(srcString string) []string {
	return strings.Split(srcString, ",")
}
