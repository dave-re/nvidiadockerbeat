package status

type NvidiaStatus struct {
	Devices []DeviceStatus
}

type ClockInfo struct {
	Cores  uint
	Memory uint
}

type UtilizationInfo struct {
	GPU     uint
	Memory  uint
	Encoder uint
	Decoder uint
}

type PCIThroughputInfo struct {
	RX uint
	TX uint
}

type PCIStatusInfo struct {
	BAR1Used   uint64
	Throughput PCIThroughputInfo
}

type ECCErrorsInfo struct {
	L1Cache uint64
	L2Cache uint64
	Global  uint64
}

type MemoryInfo struct {
	GlobalUsed uint64
	ECCErrors  ECCErrorsInfo
}

type ProcessInfo struct {
	PID        uint
	Name       string
	MemoryUsed uint64
}

type DeviceStatus struct {
	Index       *uint
	Power       uint
	Temperature uint
	Utilization UtilizationInfo
	Memory      MemoryInfo
	Clocks      ClockInfo
	PCI         PCIStatusInfo
	Processes   []ProcessInfo
}
