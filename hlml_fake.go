// hlml_fake.go
//go:build fakehlml
// +build fakehlml

package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Device struct is a placeholder for the actual device.
// Define Device struct with fields like PCIID, SerialNumber, UUID, etc.
type Device struct {
	serialNumber string
	uuid         string
	pciID        uint
	pciBusID     string
	numaNode     int
	Minor        uint
	Module       uint
}

// EventSet is a fake implementation of the HLML event set
type EventSet struct{}

// Event is a fake implementation of the HLML event
type Event struct {
	Serial string
	Etype  uint64
}

type HLMLReturn int

// EventType defines the type of event
const HlmlCriticalError = 1 << 1

const (
	HLML_SUCCESS                   HLMLReturn = 0
	HLML_ERROR_UNINITIALIZED       HLMLReturn = 1
	HLML_ERROR_INVALID_ARGUMENT    HLMLReturn = 2
	HLML_ERROR_NOT_SUPPORTED       HLMLReturn = 3
	HLML_ERROR_ALREADY_INITIALIZED HLMLReturn = 5
	HLML_ERROR_NOT_FOUND           HLMLReturn = 6
	HLML_ERROR_INSUFFICIENT_SIZE   HLMLReturn = 7
	HLML_ERROR_DRIVER_NOT_LOADED   HLMLReturn = 9
	HLML_ERROR_TIMEOUT             HLMLReturn = 10
	HLML_ERROR_AIP_IS_LOST         HLMLReturn = 15
	HLML_ERROR_MEMORY              HLMLReturn = 20
	HLML_ERROR_NO_DATA             HLMLReturn = 21
	HLML_ERROR_UNKNOWN             HLMLReturn = 49
)

// FakeHlml simulates the HLML library behavior
type FakeHlml struct{}

var (
	ErrNotIntialized      = errors.New("hlml not initialized")
	ErrInvalidArgument    = errors.New("invalid argument")
	ErrNotSupported       = errors.New("not supported")
	ErrAlreadyInitialized = errors.New("hlml already initialized")
	ErrNotFound           = errors.New("not found")
	ErrInsufficientSize   = errors.New("insufficient size")
	ErrDriverNotLoaded    = errors.New("driver not loaded")
	ErrAipIsLost          = errors.New("aip is lost")
	ErrMemoryError        = errors.New("memory error")
	ErrNoData             = errors.New("no data")
	ErrUnknownError       = errors.New("unknown error")
)

var pciBasePath = prefix + "/sys/bus/pci/devices"

// Global variables holding the simulated devices
var (
	simulatedDevices         map[uint]*Device   // Access devices by index
	simulatedDevicesBySerial map[string]*Device // Access devices by serial number
)

// initializeSimulatedDevices initializes the global variable `simulatedDevices` with the specified number of devices.
func initializeSimulatedDevices(deviceCount uint) {
	simulatedDevices = make(map[uint]*Device)
	simulatedDevicesBySerial = make(map[string]*Device)

	for i := uint(0); i < deviceCount; i++ {
		// Create a new device entry
		newDevice := &Device{
			serialNumber: generateRandomSerialNumber(),
			uuid:         generateRandomUUID(),
			pciID:        0x1da31020,
			pciBusID:     fmt.Sprintf("0000:00:1f.%d", i+1), // Create unique PCI Bus IDs based on index
			numaNode:     int(i),                            // NUMA node assigned sequentially
		}

		// Store in both maps
		simulatedDevices[i] = newDevice                              // Store by index
		simulatedDevicesBySerial[newDevice.serialNumber] = newDevice // Store by serial number
	}
}

// generateRandomSerialNumber creates a string like `AN45012345` where the last 4 digits are random.
func generateRandomSerialNumber() string {
	const prefix = "AN450"
	// Generate random last four digits
	lastFourDigits := fmt.Sprintf("%04d", rand.Intn(10000)) // Random number between 0000 and 9999
	return prefix + lastFourDigits
}

// generateRandomUUID creates a string in the format `01P0-HL2080A0-15-TNBS72-05-01-02`.
func generateRandomUUID() string {
	const basePrefix = "01P0-HL2080A0-15-TNB"

	// Define possible suffixes
	suffixes := []string{"S72", "R62", "S51"}

	// Select a random suffix
	rand.Seed(time.Now().UnixNano())
	suffix := suffixes[rand.Intn(len(suffixes))]

	// Generate a random date part
	hour := fmt.Sprintf("%02d", rand.Intn(24))    // Random year between 00-23
	month := fmt.Sprintf("%02d", rand.Intn(12)+1) // Random month between 01-12
	day := fmt.Sprintf("%02d", rand.Intn(28)+1)   // Random day between 01-28

	// Construct and return the final UUID string
	return fmt.Sprintf("%s%s-%s-%s-%s", basePrefix, suffix, month, day, hour)
}

// getHlml returns the fake HLML implementation when `realhlml` build tag is not used.
func getHlml() Hlml {
	initializeSimulatedDevices(8)
	return &FakeHlml{}
}

// errorString translates the HLML return code into a Go error
func errorString(ret HLMLReturn) error {
	switch ret {
	case HLML_SUCCESS, HLML_ERROR_TIMEOUT:
		return nil
	case HLML_ERROR_UNINITIALIZED:
		return ErrNotIntialized
	case HLML_ERROR_INVALID_ARGUMENT:
		return ErrInvalidArgument
	case HLML_ERROR_NOT_SUPPORTED:
		return ErrNotSupported
	case HLML_ERROR_ALREADY_INITIALIZED:
		return ErrAlreadyInitialized
	case HLML_ERROR_NOT_FOUND:
		return ErrNotFound
	case HLML_ERROR_INSUFFICIENT_SIZE:
		return ErrInsufficientSize
	case HLML_ERROR_DRIVER_NOT_LOADED:
		return ErrDriverNotLoaded
	case HLML_ERROR_AIP_IS_LOST:
		return ErrAipIsLost
	case HLML_ERROR_MEMORY:
		return ErrMemoryError
	case HLML_ERROR_NO_DATA:
		return ErrNoData
	case HLML_ERROR_UNKNOWN:
		return ErrUnknownError
	}
	return errors.New("invalid HLML error return code")
}

// Initialize simulates the initialization of the HLML library
func (d *FakeHlml) Initialize() error {
	// Simulate a successful initialization
	return errorString(HLML_SUCCESS)
}

// Shutdown simulates the shutdown of the HLML library in the fake implementation
func (d *FakeHlml) Shutdown() error {
	// Simulate a successful shutdown
	return errorString(HLML_SUCCESS)
}

func (d *FakeHlml) GetDeviceTypeName() (string, error) {
	var deviceType string

	fmt.Println("pciBasePath", pciBasePath)
	fmt.Println("filepath", filepath.Join(pciBasePath))
	fmt.Println("info")
	err := filepath.Walk(pciBasePath, func(path string, info os.FileInfo, err error) error {
		log.Println(pciBasePath, info.Name())
		if err != nil {
			return fmt.Errorf("error accessing file path %q", path)
		}
		if info.IsDir() {
			log.Println("Not a device, continuing")
			return nil
		}
		// Retrieve vendor for the device
		vendorID, err := readIDFromFile(pciBasePath, info.Name(), "vendor")
		if err != nil {
			return fmt.Errorf("get vendor: %w", err)
		}

		// Habana vendor id is "1da3".
		if vendorID != "1da3" {
			return nil
		}

		deviceID, err := readIDFromFile(pciBasePath, info.Name(), "device")
		if err != nil {
			return fmt.Errorf("get device info: %w", err)
		}

		deviceType, err = getDeviceName(deviceID)
		if err != nil {
			return fmt.Errorf("get device name: %w", err)
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return deviceType, nil
}

// DeviceCount simulates the retrieval of the number of Habana devices in the system
func (d *FakeHlml) DeviceCount() (uint, error) {
	// Simulate having 4 devices in the system and return success
	const simulatedDeviceCount uint = 8
	return simulatedDeviceCount, errorString(HLML_SUCCESS)
}

// DeviceHandleBySerial simulates getting a handle to a particular device by serial number
func (d *FakeHlml) DeviceHandleBySerial(serial string) (*Device, error) {
	// Check if the device with the given serial number exists
	if device, found := simulatedDevicesBySerial[serial]; found {
		return device, nil
	}

	// Return an error if the device is not found
	return nil, errors.New("could not find device with serial number")
}
func (d *FakeHlml) NewEventSet() *EventSet {
	// In the fake implementation, we simply return an empty EventSet struct
	return &EventSet{}
}

func (d *FakeHlml) DeleteEventSet(es *EventSet) {
	// In the fake implementation, we do nothing
}

// func RegisterEventForDevice(es EventSet, event int, uuid string) error {
func (d *FakeHlml) RegisterEventForDevice(es *EventSet, event int, uuid string) error {
	// In the fake implementation, we return success
	return errorString(HLML_SUCCESS)
}

func (d *FakeHlml) WaitForEvent(es *EventSet, timeout int) (*Event, error) {
	// In the fake implementation, we return a fake event
	return &Event{}, errorString(HLML_SUCCESS)
}

// DeviceHandleByIndex simulates getting a handle to a device by its index
func (d *FakeHlml) DeviceHandleByIndex(index uint) (Device, error) {
	// Check if the device with the given index exists
	if device, found := simulatedDevices[index]; found {
		return *device, nil
	} else {
		// Return an error if the device is not found
		return Device{}, errors.New("could not find device with index")
	}
}

// GetCriticalErrorCode returns a simulated critical error code
func (d *FakeHlml) HlmlCriticalError() uint64 {
	return 1 << 1 // fake value for HlmlCriticalError (same as #define HLML_EVENT_CRITICAL_ERR (1 << 1))
}

// MinorNumber simulates returning the Minor number in the fake implementation
func (d Device) MinorNumber() (uint, error) {
	// Simulate returning a minor number (hardcoded or configurable in the fake struct)
	return d.Minor, nil
}

// ModuleID simulates returning the ModuleID in the fake implementation
func (d Device) ModuleID() (uint, error) {
	// Simulate returning a module ID (hardcoded or configurable in the fake struct)
	return d.Module, nil
}

// getDeviceName returns the name of the device based on the device ID
func getDeviceName(deviceID string) (string, error) {
	goya := []string{"0001"}
	// Gaudi family includes Gaudi 1 and Guadi 2
	gaudi := []string{"1000", "1001", "1010", "1011", "1020", "1030", "1060", "1061", "1062"}
	greco := []string{"0020", "0030"}

	switch {
	case checkFamily(goya, deviceID):
		return "goya", nil
	case checkFamily(gaudi, deviceID):
		return "gaudi", nil
	case checkFamily(greco, deviceID):
		return "greco", nil
	default:
		return "", errors.New("no habana devices on the system")
	}
}

func checkFamily(family []string, id string) bool {
	for _, m := range family {
		if strings.HasSuffix(id, m) {
			return true
		}
	}
	return false
}

func readIDFromFile(basePath string, deviceAddress string, property string) (string, error) {
	data, err := os.ReadFile(filepath.Join(basePath, deviceAddress, property))
	if err != nil {
		return "", fmt.Errorf("could not read %s for device %s: %w", property, deviceAddress, err)
	}
	id := strings.Trim(string(data[2:]), "\n")
	return id, nil
}

func (d *Device) PCIID() (uint, error) {
	// Return the PCI ID of the device
	if &d.pciID == nil {
		return 0, errors.New("PCIID not available")
	}
	return d.pciID, nil
}

func (d *Device) SerialNumber() (string, error) {
	// Return the Serial Number of the device
	if d.serialNumber == "" {
		return "", errors.New("SerialNumber not available")
	}
	return d.serialNumber, nil
}

func (d *Device) UUID() (string, error) {
	// Return the UUID of the device
	if d.uuid == "" {
		return "", errors.New("UUID not available")
	}
	return d.uuid, nil
}

func (d *Device) PCIBusID() (string, error) {
	// Return the PCI Bus ID of the device
	if d.pciBusID == "" {
		return "", errors.New("PCIBusID not available")
	}
	return d.pciBusID, nil
}

// NumaNode returns the Numa affinity of the device or nil is no affinity.
func (d Device) NumaNode() (*uint, error) {
	busID, err := d.PCIBusID()
	if err != nil {
		return nil, err
	}

	b, err := os.ReadFile(fmt.Sprintf(pciBasePath+"/%s/numa_node", strings.ToLower(busID)))
	if err != nil {
		// report nil if NUMA support isn't enabled
		return nil, nil
	}
	node, err := strconv.ParseInt(string(bytes.TrimSpace(b)), 10, 8)
	if err != nil {
		return nil, fmt.Errorf("%v: %v", errors.New("failed to retrieve CPU affinity"), err)
	}
	if node < 0 {
		return nil, nil
	}

	numaNode := uint(node)
	return &numaNode, nil
}
