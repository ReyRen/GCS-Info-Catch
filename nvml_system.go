package main

import (
	pb "GCS-Info-Catch/proto"
	"errors"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"log"
)

// return value 0 is success
func (g *GCSInfoCatchServer) NvmlUtilizationRate(req *pb.NvmlInfoReuqestMsg, stream pb.GcsInfoCatchServiceDocker_NvmlUtilizationRateServer) error {
	log.Printf("Get GRPC requect, Type is %v\n", req.GetType())
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		log.Printf("Unable to initialize NVML:%v\n", nvml.ErrorString(ret))
		return errors.New(nvml.ErrorString(ret))
	}
	log.Println("nvml.Init() ok")
	defer func() {
		ret := nvml.Shutdown()
		if ret != nvml.SUCCESS {
			log.Printf("Unable to shutdown NVML:%v\n", nvml.ErrorString(ret))
			return
		}
	}()
	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		log.Printf("Unable to get device count:%v\n", nvml.ErrorString(ret))
		return errors.New(nvml.ErrorString(ret))
	}

	var indexID []int32
	var utilizationRate []uint32
	var memRate []uint32
	var temperature []uint32
	var occupied []uint32

	for i := 0; i < count; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			log.Printf("Unable to get device at index:%v\n", nvml.ErrorString(ret))
			indexID = append(indexID, 99999) // 99999表示有异常
			continue
		}
		//GPU 序列加入
		indexID = append(indexID, int32(i))
		log.Printf("GPUIndex get %v\n", indexID)
		//GPU 利用率加入
		rate, ret := device.GetUtilizationRates()
		if ret != nvml.SUCCESS {
			log.Printf("Unable to get device GetUtilizationRates at index:%v\n", nvml.ErrorString(ret))
			utilizationRate = append(utilizationRate, 99999)
			memRate = append(memRate, 99999)
			continue
		}
		utilizationRate = append(utilizationRate, rate.Gpu)
		memRate = append(memRate, rate.Memory)
		log.Printf("utilizationRate get %v memRate get %v\n", utilizationRate, memRate)
		//GPU 温度加入
		temp, ret := device.GetTemperature(nvml.TEMPERATURE_GPU)
		if ret != nvml.SUCCESS {
			log.Printf("Unable to get device GetTemperature at index:%v\n", nvml.ErrorString(ret))
			temperature = append(utilizationRate, 99999)
			continue
		}
		temperature = append(utilizationRate, temp)
		log.Printf("temperature get %v\n", temperature)
		//occupied情况
		process, ret := device.GetComputeRunningProcesses()
		if ret != nvml.SUCCESS {
			log.Printf("Unable to get device GetComputeRunningProcesses at index:%v\n", nvml.ErrorString(ret))
			occupied = append(occupied, 99999)
			continue
		}
		if process != nil {
			occupied = append(occupied, 1) // 1表示占用了
		}
		log.Printf("occupied get %v\n", occupied)
	}

	err := stream.Send(&pb.NvmlInfoRespondMsg{
		IndexID:         indexID,
		UtilizationRate: utilizationRate,
		MemRate:         memRate,
		Temperature:     temperature,
		Occupied:        occupied,
	})
	if err != nil {
		log.Printf("Stream send error:%v", err)
		return errors.New(nvml.ErrorString(nvml.ERROR_UNKNOWN))
	}
	log.Println("grpc stream send ok")
	return nil
}
