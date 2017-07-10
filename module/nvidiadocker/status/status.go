package status

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"

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

type nvidiaStatus struct {
	Devices []common.MapStr `json:"Devices"`
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
// func (m *MetricSet) Fetch() (common.MapStr, error) {

// 	resp, err := http.Get(fmt.Sprintf("%s/v1.0/gpu/status/json", m.apiURL))
// 	if err != nil {
// 		return nil, err
// 	}
// 	bytes, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	data := common.MapStr{}
// 	if err := json.Unmarshal(bytes, &data); err != nil {
// 		return nil, err
// 	}

// 	event := common.MapStr{}
// 	if devices, ok := data["Devices"].([]interface{}); ok {
// 		for i, device := range devices {
// 			event.Put(fmt.Sprintf("device%d", i), device)
// 		}
// 	}

// 	return event, nil
// }

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

func (m *MetricSet) fetchFromContainers(apiContainers []docker.APIContainers, gpuDevices []common.MapStr) ([]common.MapStr, error) {
	events := make([]common.MapStr, 0, len(apiContainers))
	for _, apiContainer := range apiContainers {
		if container, err := m.dockerClient.InspectContainer(apiContainer.ID); err == nil {
			event := fetchFromContainer(container, gpuDevices)
			if len(event["Devices"].(common.MapStr)) > 0 {
				events = append(events, event)
			}
		}
	}
	return events, nil
}

func getGPUDeviceStatus(apiURL string) ([]common.MapStr, error) {
	resp, err := http.Get(fmt.Sprintf("%s/v1.0/gpu/status/json", apiURL))
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	status := nvidiaStatus{}
	if err := json.Unmarshal(bytes, &status); err != nil {
		return nil, err
	}

	return status.Devices, nil
}

func fetchFromContainer(container *docker.Container, gpuDevices []common.MapStr) common.MapStr {
	gpuDevicesLen := len(gpuDevices)
	event := common.MapStr{
		"ContainerName": container.Name,
		"Devices":       common.MapStr{},
	}
	for _, device := range container.HostConfig.Devices {
		if findStrs := nvidiaDeviceRegexp.FindStringSubmatch(device.PathOnHost); findStrs != nil && len(findStrs) == 2 {
			if nvidiaIndex, err := strconv.ParseInt(findStrs[1], 10, 64); err == nil {
				if int(nvidiaIndex) < gpuDevicesLen {
					(event["Devices"].(common.MapStr)).Put(fmt.Sprintf("%d", nvidiaIndex), gpuDevices[nvidiaIndex])
				}
			}
		}
	}

	return event

}
