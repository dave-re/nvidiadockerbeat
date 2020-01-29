package status

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	docker "github.com/fsouza/go-dockerclient"
)

const (
	nvidiaRuntimeName          = "nvidia"
	nvidiaVisibleDevicesENVKey = "NVIDIA_VISIBLE_DEVICES"
)

var (
	nvidiaDeviceRegexp = regexp.MustCompile("^/dev/nvidia([0-9]+)$")
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("nvidiadocker", "status", New); err != nil {
		panic(err)
	}
}

type (
	// MetricSet type defines all fields of the MetricSet
	// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
	// additional entries. These variables can be used to persist data or configuration between
	// multiple fetch calls.
	MetricSet struct {
		mb.BaseMetricSet
		dockerClient *docker.Client
	}

	ContainerStatus struct {
		devices []*DeviceStatus
	}

	config struct {
		DockerEndpoint string `config:"dockerendpoint"`
	}
)

func (c *ContainerStatus) AddDevice(device *DeviceStatus) {
	c.devices = append(c.devices, device)
}

func (c *ContainerStatus) GPUSum() uint {
	return c.PropSum(func(device *DeviceStatus) uint {
		return device.Utilization.GPU
	})
}

func (c *ContainerStatus) GPUMemorySum() uint {
	return c.PropSum(func(device *DeviceStatus) uint {
		return device.Utilization.Memory
	})
}

func (c *ContainerStatus) TemperatureAverage() float64 {
	return c.PropAverage(func(device *DeviceStatus) uint {
		return device.Temperature
	})
}

func (c *ContainerStatus) PropSum(getPropFunc func(device *DeviceStatus) uint) uint {
	var total uint
	for _, device := range c.devices {
		total += getPropFunc(device)
	}
	return total
}

func (c *ContainerStatus) PropAverage(getPropFunc func(device *DeviceStatus) uint) float64 {
	if len(c.devices) == 0 {
		return 0
	}

	var total uint
	for _, device := range c.devices {
		total += getPropFunc(device)
	}
	return float64(total) / float64(len(c.devices))
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	cfg := config{
		DockerEndpoint: "",
	}

	if err := base.Module().UnpackConfig(&cfg); err != nil {
		return nil, err
	}

	dockerClient, err := docker.NewClient(cfg.DockerEndpoint)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		dockerClient:  dockerClient,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	apiContainers, err := m.dockerClient.ListContainers(docker.ListContainersOptions{})
	if err != nil {
		return nil, err
	}

	if len(apiContainers) == 0 {
		return []common.MapStr{}, nil
	}

	output, err := execNvidiaSMICommand()
	if err != nil {
		return nil, err
	}

	gpuDevices, err := getGPUDeviceStatus(output)
	if err != nil {
		return nil, err
	}

	return m.fetchFromContainers(apiContainers, gpuDevices)
}

func (m *MetricSet) fetchFromContainers(apiContainers []docker.APIContainers, gpuDevices []DeviceStatus) ([]common.MapStr, error) {
	allEvents := make([]common.MapStr, 0, len(apiContainers))
	for _, apiContainer := range apiContainers {
		if container, err := m.dockerClient.InspectContainer(apiContainer.ID); err == nil {
			event := fetchFromContainer(container, gpuDevices)
			allEvents = append(allEvents, event)
		}
	}
	return allEvents, nil
}

func getGPUDeviceStatus(nvidiaSmiRunOutput string) ([]DeviceStatus, error) {
	lines := strings.Split(strings.TrimSpace(nvidiaSmiRunOutput), "\n")
	deviceStatuses := make([]DeviceStatus, 0, len(lines))
	for _, line := range lines {
		contents := strings.Split(line, ",")
		if len(contents) != 5 {
			continue
		}

		index, err := strconv.ParseUint(strings.TrimSpace(contents[0]), 10, 64)
		if err != nil {
			return nil, err
		}

		gpuUtil, err := strconv.ParseUint(strings.TrimSpace(contents[1]), 10, 64)
		if err != nil {
			return nil, err
		}

		memTotal, err := strconv.ParseFloat(strings.TrimSpace(contents[2]), 10)
		if err != nil {
			return nil, err
		}

		memUsed, err := strconv.ParseFloat(strings.TrimSpace(contents[3]), 10)
		if err != nil {
			return nil, err
		}

		temperature, err := strconv.ParseUint(strings.TrimSpace(contents[4]), 10, 64)
		if err != nil {
			return nil, err
		}

		deviceStatuses = append(deviceStatuses, DeviceStatus{
			Index:       toUintP(uint(index)),
			Temperature: uint(temperature),
			Utilization: UtilizationInfo{
				GPU:    uint(gpuUtil),
				Memory: uint((memUsed / memTotal) * 100.0),
			},
		})

	}
	return deviceStatuses, nil
}

func execNvidiaSMICommand() (string, error) {
	outputBytes, err := exec.Command("/usr/bin/nvidia-smi",
		"--query-gpu=index,utilization.gpu,memory.total,memory.used,temperature.gpu",
		"--format=csv,noheader,nounits",
	).Output()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s", outputBytes), nil
}

func fetchFromContainer(container *docker.Container, gpuDevices []DeviceStatus) common.MapStr {
	var (
		gpuDevicesLen   = len(gpuDevices)
		containerID     = container.ID
		containerName   = strings.TrimPrefix(container.Name, "/")
		containerLabels = container.Config.Labels
		event           = common.MapStr{
			"containerid":   containerID,
			"containername": containerName,
			"labels":        containerLabels,
		}
		cStatus = &ContainerStatus{}
	)

	deviceIndices := getNvidiaVisibleDevices(container.Config.Env)
	for _, deviceIndex := range deviceIndices {
		if deviceIndex < gpuDevicesLen {
			cStatus.AddDevice(&gpuDevices[deviceIndex])
		}
	}

	event["device"] = common.MapStr{
		"Utilization": common.MapStr{
			"GPU":    cStatus.GPUSum(),
			"Memory": cStatus.GPUMemorySum(),
		},
		"Temperature": cStatus.TemperatureAverage(),
	}
	return event
}

func toUintP(val uint) *uint {
	return &val
}

func getNvidiaVisibleDevices(env []string) []int {
	deviceIndices := make([]int, 0, 8)
	for _, envStr := range env {
		if strings.HasPrefix(envStr, fmt.Sprintf("%s=", nvidiaVisibleDevicesENVKey)) {
			splitEnvStrs := strings.Split(envStr, "=")
			if len(splitEnvStrs) == 2 {
				for _, deviceIndexStr := range strings.Split(splitEnvStrs[1], ",") {
					if deviceIndex, err := strconv.ParseInt(deviceIndexStr, 10, 64); err == nil {
						deviceIndices = append(deviceIndices, int(deviceIndex))
					}
				}
			}
		}
	}
	return deviceIndices
}
