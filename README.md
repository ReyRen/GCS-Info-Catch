# GCS-Info-Catch
RPC used by GCS
The role is GCS worker set up in each work nodes

subject to https://github.com/ReyRen/GCS

https://www.cnblogs.com/niuben/p/14212878.html

https://grpc.io/docs/protoc-installation/
https://grpc.io/docs/languages/go/quickstart/

protoc -I proto/ --go_out=./proto --go_opt=paths=source_relative --go-grpc_opt=require_unimplemented_servers=false --go-grpc_out=./proto --go-grpc_opt=paths=source_relative proto/helloworld.proto 