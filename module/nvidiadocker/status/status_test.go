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
	}, gpuDevices)

	fmt.Println(events)

	if len(events) != 2 {
		t.Fatal("no events")
	}
}
