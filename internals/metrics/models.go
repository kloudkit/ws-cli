package metrics

type CPUStats struct {
	UsageSeconds     float64
	UserSeconds      float64
	SystemSeconds    float64
	ThrottledPeriods uint64
	ThrottledSeconds float64
	TotalPeriods     uint64
}

type MemoryStats struct {
	UsageBytes     uint64
	LimitBytes     uint64
	RSSBytes       uint64
	CacheBytes     uint64
	SwapBytes      uint64
	SwapLimitBytes uint64
	AnonBytes      uint64
	KernelBytes    uint64
	SlabBytes      uint64
	OOMEvents      uint64
	OOMKillEvents  uint64
	MaxEvents      uint64
}

type PIDStats struct {
	Current uint64
	Limit   uint64
}

type IOStats struct {
	ReadBytesTotal  uint64
	WriteBytesTotal uint64
	ReadOpsTotal    uint64
	WriteOpsTotal   uint64
}

type PressureStats struct {
	CPUWaitingSeconds    float64
	CPUStalledSeconds    float64
	MemoryWaitingSeconds float64
	MemoryStalledSeconds float64
	IOWaitingSeconds     float64
	IOStalledSeconds     float64
}

type NetworkStats struct {
	ReceiveBytesTotal    uint64
	TransmitBytesTotal   uint64
	ReceivePacketsTotal  uint64
	TransmitPacketsTotal uint64
	ReceiveErrorsTotal   uint64
	TransmitErrorsTotal  uint64
}

type SocketStats struct {
	TCPEstablished uint64
	TCPListen      uint64
	UDP            uint64
}

type DiskStats struct {
	UsageBytes uint64
	LimitBytes uint64
}

type FileDescriptorStats struct {
	Open  uint64
	Limit uint64
}

type GPUStats struct {
	Available          bool
	UtilizationRatio   float64
	MemoryUsedBytes    uint64
	MemoryTotalBytes   uint64
	TemperatureCelsius float64
	PowerWatts         float64
}
