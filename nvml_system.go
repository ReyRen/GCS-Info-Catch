package main

import (
	pb "GCS-Info-Catch/proto"
	"errors"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"log"
	"strconv"
)

// return value 0 is success
func (g *GCSInfoCatchServer) NvmlUtilizationRate(req *pb.NvmlInfoReuqestMsg, stream pb.GcsInfoCatchServiceDocker_NvmlUtilizationRateServer) error {
	log.Printf("Get GRPC request, indexId is %v\n", req.GetIndexID())
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
	/*count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		log.Printf("Unable to get device count:%v\n", nvml.ErrorString(ret))
		return errors.New(nvml.ErrorString(ret))
	}*/

	var indexID []int32
	var utilizationRate []uint32
	var memRate []uint64
	var temperature []uint32
	var occupied []uint32

	//handle "," index
	index_arr := stringTrimHandler(req.GetIndexID())

	for _, gpuIndex := range index_arr {
		gpuIndexInt, _ := strconv.Atoi(gpuIndex)
		device, ret := nvml.DeviceGetHandleByIndex(gpuIndexInt)
		if ret != nvml.SUCCESS {
			log.Printf("Unable to get device at index:%v\n", nvml.ErrorString(ret))
			indexID = append(indexID, 99999) // 99999表示有异常
			continue
		}
		//GPU 序列加入
		indexID = append(indexID, int32(gpuIndexInt))
		//GPU 利用率加入
		rate, ret := device.GetUtilizationRates()
		if ret != nvml.SUCCESS {
			log.Printf("Unable to get device GetUtilizationRates at index:%v\n", nvml.ErrorString(ret))
			utilizationRate = append(utilizationRate, 99999)
			continue
		}
		utilizationRate = append(utilizationRate, rate.Gpu)
		memUsed, ret := device.GetMemoryInfo()
		if ret != nvml.SUCCESS {
			log.Printf("Unable to get device memory used at index:%v\n", nvml.ErrorString(ret))
			memRate = append(memRate, 99999)
			continue
		}
		memRate = append(memRate, memUsed.Used*100/memUsed.Total)
		//GPU 温度加入
		temp, ret := device.GetTemperature(nvml.TemperatureSensors(0))
		if ret != nvml.SUCCESS {
			log.Printf("Unable to get device GetTemperature at index:%v\n", nvml.ErrorString(ret))
			temperature = append(utilizationRate, 99999)
			continue
		}
		temperature = append(utilizationRate, temp)
		//occupied情况
		process, ret := device.GetComputeRunningProcesses()
		if ret != nvml.SUCCESS {
			log.Printf("Unable to get device GetComputeRunningProcesses at index:%v\n", nvml.ErrorString(ret))
			occupied = append(occupied, 99999)
			continue
		}
		if len(process) > 0 {
			occupied = append(occupied, 1) // 1表示占用了
		} else {
			occupied = append(occupied, 0) //表示未占用
		}
	}
	log.Printf("GPUIndex get %v\n", indexID)
	log.Printf("utilizationRate get %v memRate get %v\n", utilizationRate, memRate)
	//log.Printf("temperature get %v\n", temperature)
	log.Printf("occupied get %v\n", occupied)
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
