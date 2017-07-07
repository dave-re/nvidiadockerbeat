package status

import (
	"encoding/json"
	"fmt"
	"net/http"

	"io/ioutil"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
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
	apiURL string
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	config := struct {
		APIURL string `config:"apiurl"`
	}{
		APIURL: "",
	}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		apiURL:        config.APIURL,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() (common.MapStr, error) {
	resp, err := http.Get(fmt.Sprintf("%s/v1.0/gpu/status/json", m.apiURL))
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data := common.MapStr{}
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, err
	}

	event := common.MapStr{}
	if devices, ok := data["Devices"].([]interface{}); ok {
		for i, device := range devices {
			event.Put(fmt.Sprintf("device%d", i), device)
		}
	}

	return event, nil
}
