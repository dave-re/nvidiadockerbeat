package status

import (
	"fmt"
	"log"
	"reflect"
	"testing"

	docker "github.com/fsouza/go-dockerclient"
)

func TestNvidiaDeviceRegexp(t *testing.T) {
	testDatas := []struct {
		DeviceName  string
		DeviceIndex string
		Matched     bool
	}{
		{
			"/dev/nvidia0",
			"0",
			true,
		},
		{
			"/dev/nvidia12",
			"12",
			true,
		},
		{
			"/dev/test0",
			"",
			false,
		},
		{
			"/nvidia0",
			"",
			false,
		},
		{
			"/dev/nvidia",
			"",
			false,
		},
	}

	for _, testData := range testDatas {
		findStrs := nvidiaDeviceRegexp.FindStringSubmatch(testData.DeviceName)
		if (findStrs != nil) != testData.Matched {
			t.Fatal("not matched")
		}

		if testData.Matched {
			if !reflect.DeepEqual([]string{testData.DeviceName, testData.DeviceIndex}, findStrs) {
				t.Fatal("not matched")
			}
		}

	}
}

func TestFetchFromContainer(t *testing.T) {
	gpuDevices := []DeviceStatus{
		{
			Index:       toUintP(0),
			Temperature: 15,
			Utilization: UtilizationInfo{
				GPU:    10,
				Memory: 10,
			},
		},
		{
			Index:       toUintP(1),
			Temperature: 14,
			Utilization: UtilizationInfo{
				GPU:    12,
				Memory: 6,
			},
		},
		{
			Index:       toUintP(2),
			Temperature: 48,
			Utilization: UtilizationInfo{
				GPU:    30,
				Memory: 40,
			},
		},
		{
			Index:       toUintP(3),
			Temperature: 20,
			Utilization: UtilizationInfo{
				GPU:    12,
				Memory: 14,
			},
		},
	}

	event := fetchFromContainer(&docker.Container{
		ID:   "id1",
		Name: "name1",
		HostConfig: &docker.HostConfig{
			Devices: []docker.Device{
				{
					PathOnHost:      "/dev/nvidia0",
					PathInContainer: "/dev/nvidia0",
				},
				{
					PathOnHost:      "/dev/nvidia1",
					PathInContainer: "/dev/nvidia1",
				},
			},
		},
		Config: &docker.Config{
			Labels: map[string]string{
				"com.kakaobrain.cloud.agent.id":       "4e3bb646-c7ff-4807-8295-daccfdbc5a34-S1",
				"com.kakaobrain.cloud.framework.id":   "ca766152-aa55-425f-b6fc-b84319732915-0000",
				"com.kakaobrain.cloud.framework.name": "__deepcloud-dev__",
				"com.kakaobrain.cloud.server.name":    "dave.go-172.28.11.37:1028",
				"com.nvidia.build.id":                 "20511715",
				"com.nvidia.build.ref":                "8cbbe3f50991afed6055bb714f79783fab77af54",
				"com.nvidia.cuda.version":             "8.0.61",
				"com.nvidia.cudnn.version":            "5.1.10",
				"com.nvidia.volumes.needed":           "nvidia_driver",
				"maintainer":                          "NVIDIA CORPORATION <cudatools@nvidia.com>",
			},
		},
	}, gpuDevices)

	fmt.Println(event.StringToPrint())

	// if len(events) != 2 {
	// 	t.Fatal("no events")
	// }
}

func TestGetGPUDeviceStatus(t *testing.T) {
	output := `0, 66, 47, 27
	1, 10, 14, 25
	2, 37, 0, 33
	3, 0, 0, 22
	4, 0, 0, 16
	5, 44, 20, 36
	6, 0, 0, 17
	7, 20, 3, 27
`
	predictDevicesStatus := []DeviceStatus{
		{
			Index: toUintP(0),
			Utilization: UtilizationInfo{
				GPU:    66,
				Memory: 47,
			},
			Temperature: 27,
		},
		{
			Index: toUintP(1),
			Utilization: UtilizationInfo{
				GPU:    10,
				Memory: 14,
			},
			Temperature: 25,
		},
		{
			Index: toUintP(2),
			Utilization: UtilizationInfo{
				GPU:    37,
				Memory: 0,
			},
			Temperature: 33,
		},
		{
			Index: toUintP(3),
			Utilization: UtilizationInfo{
				GPU:    0,
				Memory: 0,
			},
			Temperature: 22,
		},
		{
			Index: toUintP(4),
			Utilization: UtilizationInfo{
				GPU:    0,
				Memory: 0,
			},
			Temperature: 16,
		},
		{
			Index: toUintP(5),
			Utilization: UtilizationInfo{
				GPU:    44,
				Memory: 20,
			},
			Temperature: 36,
		},
		{
			Index: toUintP(6),
			Utilization: UtilizationInfo{
				GPU:    0,
				Memory: 0,
			},
			Temperature: 17,
		},
		{
			Index: toUintP(7),
			Utilization: UtilizationInfo{
				GPU:    20,
				Memory: 3,
			},
			Temperature: 27,
		},
	}
	devicesStatus, err := getGPUDeviceStatus(output)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(devicesStatus, predictDevicesStatus) {
		t.Fatal("failed")
	}
}

func TestExecNvidiaSMICommand(t *testing.T) {
	output, err := execNvidiaSMICommand()
	if err != nil {
		t.Fatal(err)
	}
	log.Println(output)
}
