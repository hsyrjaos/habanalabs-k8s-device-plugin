package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// ResourceManager interface
type ResourceManager interface {
	Devices() ([]*pluginapi.Device, error)
}

// DeviceManager string devType: GOYA / GAUDI
type DeviceManager struct {
	log     *slog.Logger
	devType string
}

// NewDeviceManager Init Manager
func NewDeviceManager(log *slog.Logger, devType string) *DeviceManager {
	return &DeviceManager{log: log, devType: devType}
}

// Devices Get Habana Device
func (dm *DeviceManager) Devices() ([]*pluginapi.Device, error) {
	hlmlWrapper := getHLMLWrapper() // Choose real or dummy implementation

	NumOfDevices, err := hlmlWrapper.DeviceCount()
	if err != nil {
		return nil, err
	}

	var devs []*pluginapi.Device

	dm.log.Info("Discovering devices...")
	for i := uint(0); i < NumOfDevices; i++ {
		newDevice, err := hlmlWrapper.DeviceHandleByIndex(i)
		if err != nil {
			return nil, err
		}

		pciID, err := newDevice.PCIID()
		if err != nil {
			return nil, err
		}

		serial, err := newDevice.SerialNumber()
		if err != nil {
			return nil, err
		}

		uuid, err := newDevice.UUID()
		if err != nil {
			return nil, err
		}

		pciBusID, _ := newDevice.PCIBusID()
		dID := fmt.Sprintf("%x", pciID)
		dm.log.Info(
			"Device found",
			"device", strings.ToUpper(dm.devType),
			"serial", serial,
			"uuid", uuid,
			"id", dID,
			"pci_bus_id", pciBusID,
		)

		dev := pluginapi.Device{
			ID:     serial,
			Health: pluginapi.Healthy,
		}

		cpuAffinity, err := newDevice.NumaNode()
		if err != nil {
			return nil, err
		}

		if cpuAffinity != nil {
			dm.log.Info("Device cpu affinity", "id", dID, "cpu_affinity", *cpuAffinity)
			dev.Topology = &pluginapi.TopologyInfo{
				Nodes: []*pluginapi.NUMANode{{ID: int64(*cpuAffinity)}},
			}
		}
		devs = append(devs, &dev)
	}

	return devs, nil
}

func getDevice(devs []*pluginapi.Device, id string) *pluginapi.Device {
	for _, d := range devs {
		if d.ID == id {
			return d
		}
	}
	return nil
}

func watchXIDs(ctx context.Context, devs []*pluginapi.Device, xids chan<- *pluginapi.Device) {
	hlmlWrapper := getHLMLWrapper() // Choose real or dummy implementation

	eventSet := hlmlWrapper.NewEventSet()
	defer hlmlWrapper.DeleteEventSet(eventSet)

	for _, d := range devs {
		err := hlmlWrapper.RegisterEventForDevice(eventSet, EventType(hlmlWrapper.HlmlCriticalError()), d.ID)
		if err != nil {
			slog.Error("Failed registering critical event for device. Marking it unhealthy", "device_id", d.ID, "error", err)
			xids <- d
			continue
		}
	}

	// TODO: provide as flag
	healthCheckInterval := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-healthCheckInterval.C:
			e, err := hlmlWrapper.WaitForEvent(eventSet, 1000)
			if err != nil {
				slog.Error("hlml WaitForEvent failed", "error", err.Error())
				time.Sleep(2 * time.Second)
				continue
			}

			if e.Etype != hlmlWrapper.HlmlCriticalError() {
				continue
			}

			dev, err := hlmlWrapper.DeviceHandleBySerial(e.Serial)
			if err != nil {
				slog.Error("XidCriticalError: All devices will go unhealthy", "xid", e.Etype)
				// All devices are unhealthy
				for _, d := range devs {
					xids <- d
				}
				continue
			}

			uuid, err := dev.UUID()
			if err != nil || len(uuid) == 0 {
				slog.Error("XidCriticalError: All devices will go unhealthy", "xid", e.Etype)
				// All devices are unhealthy
				for _, d := range devs {
					xids <- d
				}
				continue
			}

			for _, d := range devs {
				if d.ID == uuid {
					slog.Error("XidCriticalError: the device will go unhealthy", "xid", e.Etype, "aip", d.ID)
					xids <- d
				}
			}
		}
	}
}
