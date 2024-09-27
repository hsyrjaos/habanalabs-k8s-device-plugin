// hlml_shared.go
package main

// Hlml interface defines methods for interacting with the HLML library (real or dummy).
type Hlml interface {
	Initialize() error
	Shutdown() error
	GetDeviceTypeName() (string, error)
	DeviceCount() (uint, error)
	DeviceHandleBySerial(serial string) (*Device, error)
	NewEventSet() *EventSet
	DeleteEventSet(es *EventSet)
	RegisterEventForDevice(es *EventSet, eventType int, serial string) error
	WaitForEvent(es *EventSet, timeout int) (*Event, error)
	DeviceHandleByIndex(index uint) (Device, error)
	HlmlCriticalError() uint64
}
