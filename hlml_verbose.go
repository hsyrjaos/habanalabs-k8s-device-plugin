//go:build verbose
// +build verbose

package main

import (
	"fmt"
	"time"
)

// VerboseHlml is a wrapper that provides verbose debug output for an Hlml implementation.
type VerboseHlml struct {
	impl Hlml // Embeds any existing implementation of the Hlml interface.
}

// WrapHlml creates a verbose wrapper around an existing Hlml implementation.
func getVerboseHlml(impl Hlml) Hlml {
	return &VerboseHlml{impl: impl}
}

// logWithTimestamp prints a message with a timestamp prefix.
func logWithTimestamp(message string) {
	fmt.Printf("[%s] Debug: %s\n", time.Now().Format(time.RFC3339), message)
}

// Verbose implementations of Hlml methods with detailed logging.

func (v *VerboseHlml) Initialize() error {
	logWithTimestamp("Initializing HLML library")
	err := v.impl.Initialize()
	logWithTimestamp(fmt.Sprintf("Initialize result: %v", err))
	return err
}

func (v *VerboseHlml) Shutdown() error {
	logWithTimestamp("Shutting down HLML library")
	err := v.impl.Shutdown()
	logWithTimestamp(fmt.Sprintf("Shutdown result: %v", err))
	return err
}

func (v *VerboseHlml) GetDeviceTypeName() (string, error) {
	logWithTimestamp("Getting device type name")
	name, err := v.impl.GetDeviceTypeName()
	logWithTimestamp(fmt.Sprintf("GetDeviceTypeName result: name=%s, error=%v", name, err))
	return name, err
}

func (v *VerboseHlml) DeviceCount() (uint, error) {
	logWithTimestamp("Getting device count")
	count, err := v.impl.DeviceCount()
	logWithTimestamp(fmt.Sprintf("DeviceCount result: count=%d, error=%v", count, err))
	return count, err
}

func (v *VerboseHlml) DeviceHandleBySerial(serial string) (*Device, error) {
	logWithTimestamp(fmt.Sprintf("Getting device handle by serial: %s", serial))
	device, err := v.impl.DeviceHandleBySerial(serial)
	logWithTimestamp(fmt.Sprintf("DeviceHandleBySerial result: device=%v, error=%v", device, err))
	return device, err
}

func (v *VerboseHlml) NewEventSet() *EventSet {
	logWithTimestamp("Creating new event set")
	eventSet := v.impl.NewEventSet()
	logWithTimestamp(fmt.Sprintf("NewEventSet created: %v", eventSet))
	return eventSet
}

func (v *VerboseHlml) DeleteEventSet(es *EventSet) {
	logWithTimestamp(fmt.Sprintf("Deleting event set: %v", es))
	v.impl.DeleteEventSet(es)
	logWithTimestamp("Event set deleted")
}

func (v *VerboseHlml) RegisterEventForDevice(es *EventSet, eventType int, serial string) error {
	logWithTimestamp(fmt.Sprintf("Registering event %d for device %s in event set %v", eventType, serial, es))
	err := v.impl.RegisterEventForDevice(es, eventType, serial)
	logWithTimestamp(fmt.Sprintf("RegisterEventForDevice result: error=%v", err))
	return err
}

func (v *VerboseHlml) WaitForEvent(es *EventSet, timeout int) (*Event, error) {
	logWithTimestamp(fmt.Sprintf("Waiting for event in set %v with timeout %d ms", es, timeout))
	startTime := time.Now()
	event, err := v.impl.WaitForEvent(es, timeout)
	logWithTimestamp(fmt.Sprintf("WaitForEvent completed in %v, event: %v, error: %v", time.Since(startTime), event, err))
	return event, err
}

func (v *VerboseHlml) DeviceHandleByIndex(index uint) (Device, error) {
	logWithTimestamp(fmt.Sprintf("Getting device handle by index: %d", index))
	device, err := v.impl.DeviceHandleByIndex(index)
	logWithTimestamp(fmt.Sprintf("DeviceHandleByIndex result: device=%v, error=%v", device, err))
	return device, err
}

func (v *VerboseHlml) HlmlCriticalError() uint64 {
	logWithTimestamp("Getting critical error code")
	code := v.impl.HlmlCriticalError()
	logWithTimestamp(fmt.Sprintf("HlmlCriticalError result: code=%d", code))
	return code
}
