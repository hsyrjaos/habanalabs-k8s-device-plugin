// hlml_real.go
//go:build !fakehlml
// +build !fakehlml

package main

import (
	realhlml "github.com/HabanaAI/gohlml"
)

type RealHlml struct{}

// Type aliasing for the real HLML types to match the interface types
type Device = realhlml.Device
type EventSet = realhlml.EventSet
type Event = realhlml.Event

//type EventType = realhlml.EventType

// getHlml returns the real HLML implementation when `realhlml` build tag is used.
func getHlml() Hlml {
	return &RealHlml{}
}

func (r *RealHlml) Initialize() error {
	return realhlml.Initialize()
}

func (r *RealHlml) Shutdown() error {
	return realhlml.Shutdown()
}

func (r *RealHlml) GetDeviceTypeName() (string, error) {
	return realhlml.GetDeviceTypeName()
}

func (r *RealHlml) DeviceCount() (uint, error) {
	return realhlml.DeviceCount()
}

func (r *RealHlml) DeviceHandleBySerial(serial string) (*Device, error) {
	device, err := realhlml.DeviceHandleBySerial(serial)
	return device, err
}

func (r *RealHlml) NewEventSet() *EventSet {
	eventSet := realhlml.NewEventSet()
	return &eventSet
}

func (r *RealHlml) DeleteEventSet(eventSet *EventSet) {
	realhlml.DeleteEventSet(*eventSet)
}

func (r *RealHlml) RegisterEventForDevice(eventSet *EventSet, eventType int, serial string) error {
	return realhlml.RegisterEventForDevice(*eventSet, eventType, serial)
}

func (r *RealHlml) WaitForEvent(eventSet *EventSet, timeout int) (*Event, error) {
	event, err := realhlml.WaitForEvent(*eventSet, uint(timeout))
	return &event, err
}

func (r *RealHlml) DeviceHandleByIndex(index uint) (Device, error) {
	return realhlml.DeviceHandleByIndex(index)
}

func (r *RealHlml) HlmlCriticalError() uint64 {
	return realhlml.HlmlCriticalError
}
