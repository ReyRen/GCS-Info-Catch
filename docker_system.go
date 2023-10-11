package main

import (
	pb "GCS-Info-Catch/proto"
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"io"
	"log"
	"os"
)

func (g *GCSInfoCatchServer) DockerContainerImagePull(req *pb.ImagePullRequestMsg, stream pb.GcsInfoCatchService_DockerContainerImagePullServer) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("NewClientWithOpts Error:", err.Error())
	}
	defer cli.Close()
	var out io.ReadCloser
	// 创建一个缓冲区来保存数据
	buf := make([]byte, 4096)

	for {
		out, err = cli.ImagePull(ctx, req.GetImageName(), types.ImagePullOptions{})
		if err != nil {
			log.Printf("cli.ImagePull Error:", err.Error())
			err = stream.Send(&pb.ImagePullRespondMsg{ImageResp: "image pull get error " + err.Error()})
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
			break
		}
		err = stream.Send(&pb.ImagePullRespondMsg{ImageResp: string(buf[:n])})
		if err != nil {
			log.Printf("Stream send error:%v", err)
			return err
		}
	}
	defer out.Close()

	return nil
}

func (g *GCSInfoCatchServer) DockerContainerStart(req *pb.StartRequestMsg, stream pb.GcsInfoCatchService_DockerContainerStartServer) error {
	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		//*reply = "client.NewClientWithOpts get Error:" + request
		panic(err)
	}

	defer cli.Close()

	imageName := "request"
	out, err := cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	defer out.Close()
	io.Copy(os.Stdout, out)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		User:         "root",
		ExposedPorts: nil,
		Env:          nil,
		Healthcheck:  nil,
		Image:        imageName,
		Volumes:      nil,
		WorkingDir:   "",
		//覆盖镜像中的 entrypoint 脚本，防止镜像自动退出
		Entrypoint:  []string{"/bin/bash", "-c", "tail -f /dev/null"},
		Labels:      nil,
		StopSignal:  "",
		StopTimeout: nil,
	}, &container.HostConfig{
		Binds:           nil,
		LogConfig:       container.LogConfig{},
		NetworkMode:     "",
		PortBindings:    nil,
		RestartPolicy:   container.RestartPolicy{},
		AutoRemove:      false,
		VolumeDriver:    "",
		VolumesFrom:     nil,
		ConsoleSize:     [2]uint{},
		Annotations:     nil,
		CapAdd:          nil,
		CapDrop:         nil,
		CgroupnsMode:    "",
		DNS:             nil,
		DNSOptions:      nil,
		DNSSearch:       nil,
		ExtraHosts:      nil,
		GroupAdd:        nil,
		IpcMode:         container.IPCModeHost,
		Cgroup:          "",
		Links:           nil,
		OomScoreAdj:     0,
		PidMode:         "",
		Privileged:      true,
		PublishAllPorts: false,
		ReadonlyRootfs:  false,
		SecurityOpt:     nil,
		StorageOpt:      nil,
		Tmpfs:           nil,
		UTSMode:         "",
		UsernsMode:      "",
		ShmSize:         128,
		Resources: container.Resources{
			CPUShares:            0,
			Memory:               0,
			NanoCPUs:             0,
			CgroupParent:         "",
			BlkioWeight:          0,
			BlkioWeightDevice:    nil,
			BlkioDeviceReadBps:   nil,
			BlkioDeviceWriteBps:  nil,
			BlkioDeviceReadIOps:  nil,
			BlkioDeviceWriteIOps: nil,
			CPUPeriod:            0,
			CPUQuota:             0,
			CPURealtimePeriod:    0,
			CPURealtimeRuntime:   0,
			CpusetCpus:           "",
			CpusetMems:           "",
			//Devices:              device,
			DeviceCgroupRules: nil,
			//DeviceRequests:     deviceRequest, // 这里增加 GPU
			KernelMemoryTCP:    0,
			MemoryReservation:  0,
			MemorySwap:         0,
			MemorySwappiness:   nil,
			OomKillDisable:     nil,
			PidsLimit:          nil,
			Ulimits:            nil,
			CPUCount:           0,
			CPUPercent:         0,
			IOMaximumIOps:      0,
			IOMaximumBandwidth: 0,
		},
		Mounts:        nil,
		MaskedPaths:   nil,
		ReadonlyPaths: nil,
		Init:          nil,
	}, nil, nil, "")
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	out, err = cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}
	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	return nil
}

func (g *GCSInfoCatchServer) DockerContainerDelete(req *pb.DeleteRequestMsg, stream pb.GcsInfoCatchService_DockerContainerDeleteServer) error {
	return nil
}

func (g *GCSInfoCatchServer) DockerContainerStatus(req *pb.StatusRequestMsg, stream pb.GcsInfoCatchService_DockerContainerStatusServer) error {
	/*swarm, err := cli.SwarmInspect(ctx)
	//slog.Debug("swarm information", "SWARM_JoinTokens", swarm.JoinTokens.Manager)
	slog.Debug("swarm information", "SWARM_JoinTokens", swarm.ClusterInfo.)
	*/
	/*swarmNodes, err := cli.NodeList(ctx, types.NodeListOptions{})
	for _, v := range swarmNodes {
		slog.Debug("swarm information", "HOSTNAME", v.Description.Hostname)
		slog.Debug("swarm information", "STATUS", v.Status)
		slog.Debug("swarm information", "Availability", v.Spec.Availability)
		slog.Debug("swarm information", "Role", v.Spec.Role)
		slog.Debug("swarm information", "Annotation_name", v.Spec.Annotations.Name)
	}*/

	return nil
}

func (g *GCSInfoCatchServer) DockerContainerLogs(req *pb.LogsRequestMsg, stream pb.GcsInfoCatchService_DockerContainerLogsServer) error {
	/*cli.ContainerLogs(ctx, "name", types.ContainerLogsOptions{
		ShowStdout: false,
		ShowStderr: false,
		Since:      "",
		Until:      "",
		Timestamps: false,
		Follow:     false,
		Tail:       "",
		Details:    false,
	})*/

	return nil
}

/*device := []container.DeviceMapping{
	{
		PathOnHost:        "/dev/infiniband/uverbs0",
		PathInContainer:   "",
		CgroupPermissions: "",
	},
	{
		PathOnHost:        "/dev/infiniband/uverbs1",
		PathInContainer:   "",
		CgroupPermissions: "",
	},
}*/
//capabilities := [][]string{{"gpu"}}
//deviceIds := []string{"GPU-3a335d70-3a17-6e18-bd16-5643a1d2c0ae", "GPU-1413bca2-67df-b468-5d72-3d26dbe12205"}
/*deviceRequest := []container.DeviceRequest{
	{
		Driver: "",
		Count:  2,
		//DeviceIDs:    deviceIds,
		Capabilities: capabilities,
		//Options:      {"", ""},
	},
}*/
/*device := []container.DeviceMapping{
	{
		PathOnHost:        "/dev/infiniband/uverbs0",
		PathInContainer:   "",
		CgroupPermissions: "",
	},
	{
		PathOnHost:        "/dev/infiniband/uverbs1",
		PathInContainer:   "",
		CgroupPermissions: "",
	},
}*/
