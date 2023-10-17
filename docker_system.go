package main

import (
	pb "GCS-Info-Catch/proto"
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"io"
	"log"
	"time"
)

func (g *GCSInfoCatchServer) DockerContainerRun(req *pb.ContainerRunRequestMsg,
	stream pb.GcsInfoCatchServiceDocker_DockerContainerRunServer) error {

	log.Printf("docker container run:[%v][%v][%v]\n", req.GetContainerName(), req.GetGpuIdx(), req.GetImageName())

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
		err = stream.Send(&pb.ContainerRunRespondMsg{RunResp: "IMAGE_ERROR"})
		if err != nil {
			log.Printf("Stream send error:%v", err.Error())
			return err
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
			log.Println("out.Read EOF occurred")
			out.Close()
			break
		}
		time.Sleep(2 * time.Second)
		log.Printf("%v\n", string(buf[:n]))
		err = stream.Send(&pb.ContainerRunRespondMsg{RunResp: "IMAGE_PULLING"})
		if err != nil {
			log.Printf("Stream send error:%v", err)
			return err
		}
	}

	//开始创建容器
	err = stream.Send(&pb.ContainerRunRespondMsg{RunResp: "CONTAINER_CREATING"})
	if err != nil {
		log.Printf("Stream send error:%v", err)
		return err
	}
	log.Println("container create start")
	exposedPorts := nat.PortSet{nat.Port("22/tcp"): {}}
	portBindings := []nat.PortBinding{{
		HostIP:   "",
		HostPort: "50022",
	}}

	capabilities := [][]string{{"gpu"}}
	var deviceIDs []string
	deviceRequest := []container.DeviceRequest{
		{
			DeviceIDs:    append(deviceIDs, req.GetGpuIdx()),
			Capabilities: capabilities,
		},
	}

	var mountVolume []mount.Mount
	mountData := mount.Mount{
		Type:     "bind",
		Source:   "/storage-ftp-data",
		Target:   "/storage-root",
		ReadOnly: false,
	}

	portMaps := nat.PortMap{nat.Port("22/tcp"): portBindings}
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		User:         "root",
		ExposedPorts: exposedPorts,
		Env:          nil,
		Image:        req.GetImageName(),
		Entrypoint:   []string{"/bin/bash", "-c", "tail -f /dev/null"},
	}, &container.HostConfig{
		Binds:           nil,
		LogConfig:       container.LogConfig{},
		NetworkMode:     "",
		PortBindings:    portMaps,
		RestartPolicy:   container.RestartPolicy{},
		AutoRemove:      true,
		IpcMode:         container.IPCModeHost,
		Privileged:      false,
		PublishAllPorts: false,
		ShmSize:         512,
		Resources: container.Resources{
			DeviceRequests: deviceRequest,
		},
		Mounts: append(mountVolume, mountData),
	}, nil, nil, req.GetContainerName())

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		log.Printf("container start error:%v\n", err.Error())
		return err
	}
	log.Println("container started")

	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			status, err := containerStatus(cli, ctx, resp.ID)
			if err != nil {
				log.Printf("containerStatus get error:%v\n", err.Error())
				return err
			}
			if status == "" {
				err = stream.Send(&pb.ContainerRunRespondMsg{RunResp: "CONTAINER_REMOVED"})
				if err != nil {
					log.Printf("Stream send error:%v", err)
				}
				log.Printf("containerID not found\n")
				return err
			}
			if status == "running" {
				ci, err := cli.ContainerInspect(ctx, resp.ID)
				err = stream.Send(&pb.ContainerRunRespondMsg{RunResp: "CONTAINER_RUNNING", ContainerIp: ci.NetworkSettings.IPAddress})
				if err != nil {
					log.Printf("Stream send error:%v", err)
				}
				log.Printf("container running:%v\n", resp.ID)
				return err

			}
			if status == "created" {
				err = stream.Send(&pb.ContainerRunRespondMsg{RunResp: "CONTAINER_CREATED"})
				if err != nil {
					log.Printf("Stream send error:%v", err)
				}
				log.Printf("container created:%v\n", resp.ID)
				return err
			}
			if status == "exited" {
				err = stream.Send(&pb.ContainerRunRespondMsg{RunResp: "CONTAINER_EXITED"})
				if err != nil {
					log.Printf("Stream send error:%v", err)
				}
				log.Printf("container exited:%v\n", resp.ID)
				return err
			}
			if status == "dead" {
				err = stream.Send(&pb.ContainerRunRespondMsg{RunResp: "CONTAINER_DEAD"})
				if err != nil {
					log.Printf("Stream send error:%v", err)
				}
				log.Printf("container dead:%v\n", resp.ID)
				return err
			}
			if status == "removing" {
				err = stream.Send(&pb.ContainerRunRespondMsg{RunResp: "CONTAINER_REMOVING"})
				if err != nil {
					log.Printf("Stream send error:%v", err)
				}
				log.Printf("container removing:%v\n", resp.ID)
				return err
			}
			if status == "restarting" {
				err = stream.Send(&pb.ContainerRunRespondMsg{RunResp: "CONTAINER_RESTARTING"})
				if err != nil {
					log.Printf("Stream send error:%v", err)
				}
				log.Printf("container restarting:%v\n", resp.ID)
				return err
			}
		}
	}
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

	inspect, err := cli.ContainerInspect(ctx, req.GetContainerName())
	if err != nil {
		log.Printf("ContainerInspect get error:%v\n", err.Error())
		err = stream.Send(&pb.StatusRespondMsg{StatusResp: "CONTAINER_REMOVED"})
		return err
	}
	status := inspect.State.Status
	if err != nil {
		log.Printf("containerStatus get error:%v\n", err.Error())
		return err
	}
	if status == "running" {
		log.Printf("container running:%v\n", req.GetContainerName())
		err = stream.Send(&pb.StatusRespondMsg{StatusResp: "CONTAINER_RUNNING"})
		if err != nil {
			log.Printf("Stream send error:%v", err)
			return err
		}
	}
	if status == "created" {
		err = stream.Send(&pb.StatusRespondMsg{StatusResp: "CONTAINER_CREATED"})
		if err != nil {
			log.Printf("Stream send error:%v", err)
			return err
		}
		log.Printf("container created:%v\n", req.GetContainerName())
	}
	if status == "exited" {
		err = stream.Send(&pb.StatusRespondMsg{StatusResp: "CONTAINER_EXITED"})
		if err != nil {
			log.Printf("Stream send error:%v", err)
			return err
		}
		log.Printf("container exited:%v\n", req.GetContainerName())
	}
	if status == "dead" {
		err = stream.Send(&pb.StatusRespondMsg{StatusResp: "CONTAINER_DEAD"})
		if err != nil {
			log.Printf("Stream send error:%v", err)
			return err
		}
		log.Printf("container dead:%v\n", req.GetContainerName())
	}
	if status == "removing" {
		err = stream.Send(&pb.StatusRespondMsg{StatusResp: "CONTAINER_REMOVING"})
		if err != nil {
			log.Printf("Stream send error:%v", err)
			return err
		}
		log.Printf("container removing:%v\n", req.GetContainerName())
	}
	if status == "restarting" {
		err = stream.Send(&pb.StatusRespondMsg{StatusResp: "CONTAINER_RESTARTING"})
		if err != nil {
			log.Printf("Stream send error:%v", err)
			return err
		}
		log.Printf("container restarting:%v\n", req.GetContainerName())
	}
	return nil
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
	defer out.Close()
	// 创建一个缓冲区来保存数据
	buf := make([]byte, 4096)
	for {
		out, err := cli.ContainerLogs(ctx, req.GetContainerName(), types.ContainerLogsOptions{
			ShowStdout: false,
			ShowStderr: false,
			Since:      "",
			Until:      "",
			Timestamps: false,
			Follow:     true,
			Tail:       "",
			Details:    false,
		})
		if err != nil {
			log.Printf("cli.ContainerLogs Error:", err.Error())
			err = stream.Send(&pb.LogsRespondMsg{LogsResp: "LOGS_ERROR"})
			if err != nil {
				log.Printf("Stream send error:%v", err.Error())
				return err
			}
			return err
		}

		n, err := out.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("out.Read error occurred: %v", err.Error())
				return err
			}
			// EOF
			log.Println("out.Read EOF occurred")
			break
		}
		log.Printf("%v\n", buf[:n])
		err = stream.Send(&pb.LogsRespondMsg{LogsResp: string(buf[:n])})
		if err != nil {
			log.Printf("Stream send error:%v", err)
			return err
		}
	}
	return nil
}
