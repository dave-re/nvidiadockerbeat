package status

type NvidiaStatus struct {
	Devices []DeviceStatus
}

type UtilizationInfo struct {
	GPU    uint
	Memory uint
}

type DeviceStatus struct {
	Index       *uint
	Temperature uint
	Utilization UtilizationInfo
}
