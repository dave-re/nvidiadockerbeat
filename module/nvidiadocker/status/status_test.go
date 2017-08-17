package status

import (
	"encoding/json"
	"fmt"
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
	devicesJSON := `[{"Power":13,"Temperature":15,"Utilization":{"GPU":1,"Memory":1,"Encoder":0,"Decoder":0},"Memory":{"GlobalUsed":8,"ECCErrors":{"L1Cache":null,"L2Cache":null,"Global":null}},"Clocks":{"Cores":40,"Memory":405},"PCI":{"BAR1Used":2,"Throughput":{"RX":0,"TX":0}},"Processes":null},{"Power":9,"Temperature":14,"Utilization":{"GPU":0,"Memory":0,"Encoder":0,"Decoder":0},"Memory":{"GlobalUsed":7,"ECCErrors":{"L1Cache":null,"L2Cache":null,"Global":null}},"Clocks":{"Cores":40,"Memory":405},"PCI":{"BAR1Used":2,"Throughput":{"RX":0,"TX":0}},"Processes":null},{"Power":9,"Temperature":18,"Utilization":{"GPU":0,"Memory":0,"Encoder":0,"Decoder":0},"Memory":{"GlobalUsed":7,"ECCErrors":{"L1Cache":null,"L2Cache":null,"Global":null}},"Clocks":{"Cores":40,"Memory":405},"PCI":{"BAR1Used":2,"Throughput":{"RX":0,"TX":0}},"Processes":null},{"Power":9,"Temperature":16,"Utilization":{"GPU":0,"Memory":0,"Encoder":0,"Decoder":0},"Memory":{"GlobalUsed":7,"ECCErrors":{"L1Cache":null,"L2Cache":null,"Global":null}},"Clocks":{"Cores":40,"Memory":405},"PCI":{"BAR1Used":2,"Throughput":{"RX":0,"TX":0}},"Processes":null},{"Power":9,"Temperature":20,"Utilization":{"GPU":0,"Memory":0,"Encoder":0,"Decoder":0},"Memory":{"GlobalUsed":7,"ECCErrors":{"L1Cache":null,"L2Cache":null,"Global":null}},"Clocks":{"Cores":40,"Memory":405},"PCI":{"BAR1Used":2,"Throughput":{"RX":0,"TX":0}},"Processes":null},{"Power":9,"Temperature":15,"Utilization":{"GPU":0,"Memory":0,"Encoder":0,"Decoder":0},"Memory":{"GlobalUsed":7,"ECCErrors":{"L1Cache":null,"L2Cache":null,"Global":null}},"Clocks":{"Cores":40,"Memory":405},"PCI":{"BAR1Used":2,"Throughput":{"RX":0,"TX":0}},"Processes":null},{"Power":9,"Temperature":18,"Utilization":{"GPU":0,"Memory":0,"Encoder":0,"Decoder":0},"Memory":{"GlobalUsed":7,"ECCErrors":{"L1Cache":null,"L2Cache":null,"Global":null}},"Clocks":{"Cores":40,"Memory":405},"PCI":{"BAR1Used":2,"Throughput":{"RX":0,"TX":0}},"Processes":null},{"Power":9,"Temperature":17,"Utilization":{"GPU":0,"Memory":0,"Encoder":0,"Decoder":0},"Memory":{"GlobalUsed":7,"ECCErrors":{"L1Cache":null,"L2Cache":null,"Global":null}},"Clocks":{"Cores":40,"Memory":405},"PCI":{"BAR1Used":2,"Throughput":{"RX":0,"TX":0}},"Processes":null}]`
	gpuDevices := []DeviceStatus{}
	if err := json.Unmarshal([]byte(devicesJSON), &gpuDevices); err != nil {
		t.Fatal(err)
	}

	events := fetchFromContainer(&docker.Container{
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

	fmt.Println(events)

	if len(events) != 2 {
		t.Fatal("no events")
	}
}
