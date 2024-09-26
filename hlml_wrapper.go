package main

// HLMLWrapper interface defines methods for interacting with the HLML library (real or dummy)
type HLMLWrapper interface {
	Initialize() error
	Shutdown() error
	GetDeviceTypeName() (string, error)
	DeviceCount() (uint, error)
	DeviceHandleBySerial(serial string) (*Device, error)
	NewEventSet() *EventSet
	DeleteEventSet(es *EventSet)
	RegisterEventForDevice(es *EventSet, event EventType, uuid string) error
	WaitForEvent(es *EventSet, timeout int) (*Event, error)
	DeviceHandleByIndex(index uint) (*Device, error)
	HlmlCriticalError() uint64
}

// getHLMLWrapper returns the appropriate implementation (real or dummy) based on an environment variable
func getHLMLWrapper() HLMLWrapper {
	//if strings.ToLower(os.Getenv("USE_HLML")) == "true" {
	//	return &RealHLML{} // Will be compiled only with `hlml` build tag
	//}
	return &DummyHLML{} // Dummy implementation if `hlml` build tag is not used
}
