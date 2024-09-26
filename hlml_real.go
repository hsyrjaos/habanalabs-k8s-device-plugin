// +build hlml

package main

import (
	hlml "github.com/HabanaAI/gohlml"
)

type RealHLML struct{}

func (r *RealHLML) Initialize() error {
	return hlml.Initialize()
}

func (r *RealHLML) Shutdown() error {
	return hlml.Shutdown()
}

func (r *RealHLML) GetDeviceTypeName() (string, error) {
	return hlml.GetDeviceTypeName()
}

func (r *RealHLML) DeviceCount() (uint, error) {
	return hlml.DeviceCount()
}

func (r *RealHLML) DeviceHandleBySerial(serial string) (*hlml.Device, error) {
	return hlml.DeviceHandleBySerial(serial)
}

func (r *RealHLML) NewEventSet() *hlml.EventSet {
	return hlml.NewEventSet()
}

func (r *RealHLML) DeleteEventSet(eventSet *hlml.EventSet) {
	hlml.DeleteEventSet(eventSet)
}

func (r *RealHLML) RegisterEventForDevice(eventSet *hlml.EventSet, eventType hlml.EventType, serial string) error {
	return hlml.RegisterEventForDevice(eventSet, eventType, serial)
}

func (r *RealHLML) WaitForEvent(eventSet *hlml.EventSet, timeout int) (*hlml.Event, error) {
	return hlml.WaitForEvent(eventSet, timeout)
}

func (r *RealHLML) DeviceHandleByIndex(index uint) (*hlml.Device, error) {
    return hlml.DeviceHandleByIndex(index)
}

