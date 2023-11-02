package main

import (
	pb "GCS-Info-Catch/proto"
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"io"
	"log"
	"os"
	"time"
)

func (g *GCSInfoCatchServer) DockerContainerRun(req *pb.ContainerRunRequestMsg,
	stream pb.GcsInfoCatchServiceDocker_DockerContainerRunServer) error {

	log.Printf("docker container run:[%v][%v][%v][%v][%v]\n",
		req.GetContainerName(),
		req.GetGpuIdx(),
		req.GetImageName(),
		req.GetMaster(),
		req.GetParamaters())

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("NewClientWithOpts Error:", err.Error())
		return err
	}
	defer cli.Close()
	var out io.ReadCloser
	// 创建一个缓冲区来保存数据
	buf := make([]byte, 4096)

	authConfig := registry.AuthConfig{
		Username:      DOCKER_REGISTRY_USERNAME,
		Password:      DOCKER_REGISTRY_PASSWORD,
		ServerAddress: DOCKER_REGISTRY_ADDRESS,
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		log.Printf("auth registry Marshal error:%v\n", err.Error())
		return err
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	log.Println("image pull start")

	out, err = cli.ImagePull(ctx, req.GetImageName(), types.ImagePullOptions{RegistryAuth: authStr})
	if err != nil {
		log.Printf("cli.ImagePull Error:", err.Error())
		err_stream := stream.Send(&pb.ContainerRunRespondMsg{RunResp: "IMAGE_ERROR"})
		if err_stream != nil {
			log.Printf("Stream send error:%v", err_stream.Error())
			return err_stream
		}
		return err
	}
	defer out.Close()
	for {
		n, err := out.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("out.Read error occurred: %v", err.Error())
				return err
			}
			// EOF
			log.Println("image pull EOF")
			break
		}
		time.Sleep(3 * time.Second) // 每 3 秒获取一次
		log.Printf("%v\n", string(buf[:n]))
		err_stream := stream.Send(&pb.ContainerRunRespondMsg{RunResp: "IMAGE_PULLING"})
		if err_stream != nil {
			log.Printf("Stream send error:%v", err_stream.Error())
			return err_stream
		}
	}

	//开始创建容器
	err_stream := stream.Send(&pb.ContainerRunRespondMsg{RunResp: "CONTAINER_CREATING"})
	if err_stream != nil {
		log.Printf("Stream send error:%v", err_stream.Error())
		return err_stream
	}
	log.Println("container create start")
	/*exposedPorts := nat.PortSet{nat.Port("22/tcp"): {}}
	portBindings := []nat.PortBinding{{
		HostIP:   "",
		HostPort: "50022",
	}}*/
	//portMaps := nat.PortMap{nat.Port("22/tcp"): portBindings}

	capabilities := [][]string{{"gpu"}}
	var deviceIDs []string
	deviceRequest := []container.DeviceRequest{
		{
			DeviceIDs:    append(deviceIDs, req.GetGpuIdx()),
			Capabilities: capabilities,
		},
	}

	var mountVolume []mount.Mount
	mountAll := mount.Mount{
		Type:     "bind",
		Source:   SOURCE_ALL,
		Target:   TARGET_ALL,
		ReadOnly: false,
	}
	/*mountDatasets := mount.Mount{
		Type:     "bind",
		Source:   SOURCE_DATASETS,
		Target:   TARGET_DATASETS,
		ReadOnly: false,
	}*/

	var entryPoint []string

	if req.GetMaster() {
		//是 master 执行多的命令
		entryPoint = []string{"/bin/bash", startScript, req.GetParamaters()}
	} else {
		entryPoint = []string{"/bin/bash", "-c", "service ssh start;tail -f /dev/null"}
	}
	m := make(map[string]*network.EndpointSettings)
	m["myNet"] = &network.EndpointSettings{
		NetworkID: MY_OVERLAY_NETWORK,
	}
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		User: "root",
		Tty:  true,
		//ExposedPorts: exposedPorts,
		Image:      req.GetImageName(),
		Entrypoint: entryPoint,
		Shell:      []string{"/bin/bash"},
	}, &container.HostConfig{
		Binds:     nil,
		LogConfig: container.LogConfig{},
		//PortBindings:  portMaps,
		RestartPolicy: container.RestartPolicy{},
		//AutoRemove:      true,
		IpcMode:         container.IPCModeHost,
		Privileged:      false,
		PublishAllPorts: false,
		ShmSize:         512,
		Resources: container.Resources{
			Devices: []container.DeviceMapping{{
				MY_DEVICE_INFINIBAND,
				MY_DEVICE_INFINIBAND,
				"rmw"}},
			DeviceRequests: deviceRequest,
		},
		Mounts: append(mountVolume, mountAll),
	}, &network.NetworkingConfig{m}, nil, req.GetContainerName())

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Printf("container start error:%v\n", err.Error())
		return err
	}
	log.Println("container started")
	err_stream = stream.Send(&pb.ContainerRunRespondMsg{RunResp: "CONTAINER_CREATED"})
	if err_stream != nil {
		log.Printf("Stream send error:%v", err_stream.Error())
		return err_stream
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			status, err := containerStatus(cli, ctx, resp.ID)
			if err != nil {
				log.Printf("containerStatus get error:%v\n", err.Error())
				return err
			}
			//说明containerInspect 发生了错误，大概率是没有这个 container
			if status == "" {
				err_stream := stream.Send(&pb.ContainerRunRespondMsg{RunResp: "CONTAINER_REMOVED"})
				if err_stream != nil {
					log.Printf("Stream send error:%v", err_stream.Error())
					return err_stream
				}
				log.Printf("container [%v] removed\n", req.GetContainerName())
				return nil
			} else if status == "running" {
				ci, _ := cli.ContainerInspect(ctx, resp.ID)
				err_stream := stream.Send(&pb.ContainerRunRespondMsg{RunResp: "CONTAINER_RUNNING", ContainerIp: ci.NetworkSettings.Networks[MY_OVERLAY_NETWORK].IPAddress})
				if err_stream != nil {
					log.Printf("Stream send error:%v", err_stream.Error())
					return err_stream
				}
				log.Printf("container running:%v\n", resp.ID)
				return nil
			} else {
				err_stream := stream.Send(&pb.ContainerRunRespondMsg{RunResp: "CONTAINER_UNKNOWN"})
				if err_stream != nil {
					log.Printf("Stream send error:%v", err_stream.Error())
					return err_stream
				}
				log.Printf("container unknown:%v\n", resp.ID)
				return nil
			}
		}
	}
}

func (g *GCSInfoCatchServer) DockerLogStor(ctx context.Context, req *pb.DockerLogStorReqMsg) (*pb.DockerLogStorRespMsg, error) {
	log.Printf("docker logStor:[%v][%v]\n", req.GetLogFilePath(), req.GetContainerName())

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("NewClientWithOpts Error:", err.Error())
		return &pb.DockerLogStorRespMsg{
			LogStorResp: "LOGSTOR_ERROR",
		}, err
	}
	defer cli.Close()
	var out io.ReadCloser
	// 创建一个缓冲区来保存数据
	buf := make([]byte, 4096)
	out, err = cli.ContainerLogs(ctx, req.GetContainerName(), types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: false,
		Follow:     true,
	})
	if err != nil {
		log.Printf("cli.ContainerLogs Error:", err.Error())
		return &pb.DockerLogStorRespMsg{
			LogStorResp: "LOGSTOR_ERROR",
		}, err
	}

	file, err := os.OpenFile(req.GetLogFilePath(), os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		return &pb.DockerLogStorRespMsg{
			LogStorResp: "LOGSTOR_ERROR",
		}, err
		log.Printf("openFile get error:%v\n", err)
	}
	defer file.Close()

	for {
		n, err := out.Read(buf)
		if err != nil {
			if err != io.EOF {
				return &pb.DockerLogStorRespMsg{
					LogStorResp: "LOGSTOR_ERROR",
				}, err
			}
			// EOF
			log.Println("Get log EOF")
			break
		}
		//写入文件时，使用带缓存的 *Writer
		write := bufio.NewWriter(file)
		write.WriteString(string(buf[:n]))
		//Flush将缓存的文件真正写入到文件中
		write.Flush()
	}
	return &pb.DockerLogStorRespMsg{
		LogStorResp: "LOGSTOR_OVER",
	}, nil
}

// containerStatus returns one of "created", "running", "paused", "restarting", "removing", "exited", or "dead"
// will return nil if containerID is not found
func containerStatus(c *client.Client, ctx context.Context, containerID string) (string, error) {
	ci, err := c.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", err
	}
	return ci.ContainerJSONBase.State.Status, nil
}

func (g *GCSInfoCatchServer) DockerContainerDelete(req *pb.DeleteRequestMsg, stream pb.GcsInfoCatchServiceDocker_DockerContainerDeleteServer) error {
	log.Printf("docker container delete:[%v]\n", req.GetContainName())

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("NewClientWithOpts Error:", err.Error())
		return err
	}

	err = cli.ContainerRemove(ctx, req.GetContainName(), types.ContainerRemoveOptions{
		Force: true,
	})
	if err != nil {
		log.Printf("ContainerRemove Error:", err.Error())
		return err
	}
	return nil
}

func (g *GCSInfoCatchServer) DockerContainerStatus(req *pb.StatusRequestMsg, stream pb.GcsInfoCatchServiceDocker_DockerContainerStatusServer) error {
	log.Printf("docker container status:[%v]\n", req.GetContainerName())

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("NewClientWithOpts Error:", err.Error())
		return err
	}

	status, err := containerStatus(cli, ctx, req.GetContainerName())
	if err != nil {
		log.Printf("containerStatus get error:%v\n", err.Error())
		return err
	}
	//说明containerInspect 发生了错误，大概率是没有这个 container
	if status == "" {
		err_stream := stream.Send(&pb.StatusRespondMsg{StatusResp: "CONTAINER_REMOVED"})
		if err_stream != nil {
			log.Printf("Stream send error:%v", err_stream.Error())
			return err_stream
		}
		log.Printf("container [%v] removed\n", req.GetContainerName())
		return nil
	} else if status == "running" {
		err_stream := stream.Send(&pb.StatusRespondMsg{StatusResp: "CONTAINER_RUNNING"})
		if err_stream != nil {
			log.Printf("Stream send error:%v", err_stream.Error())
			return err_stream
		}
		log.Printf("container running:%v\n", req.GetContainerName())
		return nil
	} else {
		err_stream := stream.Send(&pb.StatusRespondMsg{StatusResp: "CONTAINER_UNKNOWN"})
		if err_stream != nil {
			log.Printf("Stream send error:%v", err_stream.Error())
			return err_stream
		}
		log.Printf("container running:%v\n", req.GetContainerName())
		return nil
	}
}

func (g *GCSInfoCatchServer) DockerContainerLogs(req *pb.LogsRequestMsg, stream pb.GcsInfoCatchServiceDocker_DockerContainerLogsServer) error {
	log.Printf("docker container log:[%v]\n", req.GetContainerName())

	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("NewClientWithOpts Error:", err.Error())
		return err
	}
	defer cli.Close()
	var out io.ReadCloser
	// 创建一个缓冲区来保存数据
	buf := make([]byte, 4096)

	out, err = cli.ContainerLogs(ctx, req.GetContainerName(), types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: false,
		Follow:     true,
	})
	if err != nil {
		log.Printf("cli.ContainerLogs Error:", err.Error())
		err_stream := stream.Send(&pb.LogsRespondMsg{LogsResp: "LOGS_ERROR"})
		if err_stream != nil {
			log.Printf("Stream send error:%v", err_stream.Error())
			return err_stream
		}
		return err
	}
	for {
		n, err := out.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("out.Read error occurred: %v", err.Error())
				return err
			}
			// EOF
			log.Println("Get log EOF")
			break
		}
		//log.Printf("%v", string(buf[:n]))
		err_stream := stream.Send(&pb.LogsRespondMsg{LogsResp: string(buf[:n])})
		if err_stream != nil {
			log.Printf("Stream send error:%v", err_stream.Error())
			return err_stream
		}
	}
	return nil
}
