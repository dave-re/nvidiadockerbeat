package status

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	docker "github.com/fsouza/go-dockerclient"
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

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	apiURL       string
	dockerClient *docker.Client
}

type ContainerStatus struct {
	devices []*DeviceStatus
}

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

	config := struct {
		APIURL         string `config:"apiurl"`
		DockerEndpoint string `config:"dockerendpoint"`
	}{
		APIURL:         "",
		DockerEndpoint: "",
	}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	dockerClient, err := docker.NewClient(config.DockerEndpoint)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		apiURL:        config.APIURL,
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

	gpuDevices, err := getGPUDeviceStatus(m.apiURL)
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

func getGPUDeviceStatus(apiURL string) ([]DeviceStatus, error) {
	resp, err := http.Get(fmt.Sprintf("%s/v1.0/gpu/status/json", apiURL))
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	status := NvidiaStatus{}
	if err := json.Unmarshal(bytes, &status); err != nil {
		return nil, err
	}

	return status.Devices, nil
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

	for _, device := range container.HostConfig.Devices {
		if findStrs := nvidiaDeviceRegexp.FindStringSubmatch(device.PathOnHost); findStrs != nil && len(findStrs) == 2 {
			if nvidiaIndex, err := strconv.ParseInt(findStrs[1], 10, 64); err == nil {
				if int(nvidiaIndex) < gpuDevicesLen {
					cStatus.AddDevice(&gpuDevices[nvidiaIndex])
				}
			}
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
