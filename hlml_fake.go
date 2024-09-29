// hlml_fake.go
//go:build fakehlml
// +build fakehlml

/*
 * Copyright (c) 2024, Intel Corporation.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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
	"syscall"
	"time"

	"gopkg.in/yaml.v2"
)

// Device struct is a placeholder for the actual device.
// Define Device struct with fields like PCIID, SerialNumber, UUID, etc.
type Device struct {
	serialNumber string
	uuid         string
	pciID        string
	pciBusID     string
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

type FakeDeviceConfig struct {
	Path        string `yaml:"Path"`
	DeviceCount uint   `yaml:"DeviceCount"`
	NumaNodes   uint   `yaml:"NumaNodes"`
	PciID       string `yaml:"PciID"`
	pciBasePath string
	devBasePath string
}

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

var config FakeDeviceConfig
var prefix string

func updateConfig(yamlConfig string) error {
	// Parse the YAML string into the Config struct
	err := yaml.Unmarshal([]byte(yamlConfig), &config)
	if err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}

	config.pciBasePath = config.Path + "/sys/bus/pci/devices"
	config.devBasePath = config.Path + "/dev/accel"
	prefix = config.Path

	return nil
}

// Global variables holding the simulated devices
var (
	simulatedDevices         map[uint]*Device   // Access devices by index
	simulatedDevicesBySerial map[string]*Device // Access devices by serial number
)

// initializeSimulatedDevices initializes the global variable `simulatedDevices` with the specified number of devices.
func initializeSimulatedDevices(config FakeDeviceConfig) {
	simulatedDevices = make(map[uint]*Device)
	simulatedDevicesBySerial = make(map[string]*Device)

	// Create a new random generator instance
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomHex := rng.Intn(256)
	for i := uint(0); i < config.DeviceCount; i++ {
		// Create a new device entry
		newDevice := &Device{
			serialNumber: generateRandomSerialNumber(),
			uuid:         generateRandomUUID(),
			pciID:        config.PciID,                                     // Gaudi vendor ID and device ID
			pciBusID:     fmt.Sprintf("0000:%02x:00.0", uint(randomHex)+i), // Create unique PCI Bus IDs based on index
			Module:       i,
			Minor:        i * 2,
		}

		// Store in both maps
		simulatedDevices[i] = newDevice                              // Store by index
		simulatedDevicesBySerial[newDevice.serialNumber] = newDevice // Store by serial number
	}

	if err := createDeviceNodes(config.devBasePath, config.DeviceCount); err != nil {
		log.Fatalf("Error creating device nodes: %v", err)
	}

	if err := createSymlinkedDirectories(config.pciBasePath, config.DeviceCount, config.NumaNodes); err != nil {
		log.Fatalf("Error creating symlinked directories: %v", err)
	}
}

// generateRandomSerialNumber creates a string like `AN45012345` where the last 4 digits are random.
func generateRandomSerialNumber() string {
	const baseprefix = "FA450"
	// Generate random last four digits
	lastFourDigits := fmt.Sprintf("%04d", rand.Intn(10000)) // Random number between 0000 and 9999
	return baseprefix + lastFourDigits
}

// generateRandomUUID creates a string in the format `01P0-HL2080A0-15-TNBS72-05-01-02`.
func generateRandomUUID() string {
	const basePrefix = "01F0-AK2080E0-15-ACC"

	// Define possible suffixes
	suffixes := []string{"EL2", "EL2", "EL1"}

	// Create a new random generator instance
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Select a random suffix using the local random generator
	suffix := suffixes[rng.Intn(len(suffixes))]

	// Generate a random date part
	hour := fmt.Sprintf("%02d", rng.Intn(24))    // Random hour between 00-23
	month := fmt.Sprintf("%02d", rng.Intn(12)+1) // Random month between 01-12
	day := fmt.Sprintf("%02d", rng.Intn(28)+1)   // Random day between 01-28

	// Construct and return the final UUID string
	return fmt.Sprintf("%s%s-%s-%s-%s", basePrefix, suffix, month, day, hour)
}

// getHlml returns the fake HLML implementation when `realhlml` build tag is not used.
func getHlml() Hlml {
	yamlContent := `
Path: "/tmp/gaudi2"
DeviceCount: 8
NumaNodes: 2
PciID: "1da3:1020"
`

	// Check if FAKEACCEL_SPEC environment variable is defined
	fakeAccel := os.Getenv("FAKEACCEL_SPEC")
	if fakeAccel != "" && fakeAccel != "default" {
		log.Println("FAKEACCEL_SPEC environment variable detected, using custom Fake Device configuration")
		yamlContent = fakeAccel // Use the value from FAKEACCEL_SPEC environment variable
	}

	// Read and parse the YAML content
	err := updateConfig(yamlContent)
	if err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}

	initializeSimulatedDevices(config)
	return &FakeHlml{}
}

func createDeviceNodes(path string, count uint) error {
	// Remove the existing directory (if it exists) before creating a new one
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove existing directory %s: %v", path, err)
	}

	// Create the target directory if it does not exist
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", path, err)
	}

	// Loop to create `accel` and `accel_controlD` device nodes
	for i := 0; uint(i) < count; i++ {
		// Create accel%i device nodes
		accelName := fmt.Sprintf("%s/accel%d", path, i)
		if err := createDeviceNode(accelName, 508, uint32(i*2), syscall.S_IFCHR|0600); err != nil {
			return err
		}

		// Create accel_controlD%i device nodes
		controlName := fmt.Sprintf("%s/accel_controlD%d", path, i)
		if err := createDeviceNode(controlName, 508, uint32(i*2+1), syscall.S_IFCHR|0600); err != nil {
			return err
		}
	}

	return nil
}

// createDeviceNode creates a device node with the specified path, major, and minor number
func createDeviceNode(path string, major, minor uint32, mode uint32) error {
	dev := int((major << 8) | minor) // Combine major and minor to create a device ID
	if err := syscall.Mknod(path, mode, dev); err != nil {
		return fmt.Errorf("failed to create device node %s: %v", path, err)
	}
	return nil
}

// createSymlinkedDirectories creates symlinked directories and the target folders with files.
func createSymlinkedDirectories(path string, count uint, numaNodes uint) error {
	// Remove the existing directory (if it exists) before creating a new one
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove existing directory %s: %v", path, err)
	}

	// Create the target directory if it does not exist
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", path, err)
	}

	// Calculate how many devices per NUMA node
	devicesPerNode := count / numaNodes
	if devicesPerNode < 1 {
		devicesPerNode = 1
	}

	for i := uint(1); i <= count; i++ {
		// Get the PCI Bus ID for the current device from the simulatedDevices array
		device := simulatedDevices[i-1]

		// Create the symlink name: 0000.0a.1f.<index>
		symlinkName := fmt.Sprintf("%s/%s", path, device.pciBusID)

		// Extract the PCI root and the target directory path based on the PCI Bus ID
		pciRoot := device.pciBusID[:9] // Extract "0000:0a" from "0000:0a:1f.1"
		targetDir := fmt.Sprintf("../../../devices/pci%s/%s", pciRoot, device.pciBusID)

		// Create the absolute path of the target directory
		fullTargetPath := filepath.Join(path, targetDir)

		// Create the target directory structure if it doesn't exist
		if err := os.MkdirAll(fullTargetPath, 0755); err != nil {
			return fmt.Errorf("failed to create target directory %s: %v", fullTargetPath, err)
		}

		// Create the symlink in the path directory pointing to the target directory
		if err := os.Symlink(targetDir, symlinkName); err != nil {
			return fmt.Errorf("failed to create symlink %s -> %s: %v", symlinkName, targetDir, err)
		}

		// Determine the NUMA node for this device
		numaNode := (i - 1) / devicesPerNode

		// Create the files inside the target directory with the corresponding NUMA node value
		if err := createFilesInDirectory(fullTargetPath, i, numaNode); err != nil {
			return fmt.Errorf("failed to create files in directory %s: %v", fullTargetPath, err)
		}
	}

	return nil
}

// createFilesInDirectory creates the specified files in the given directory with the specified NUMA node.
func createFilesInDirectory(dir string, index uint, numaNode uint) error {
	// Define the file names and their contents
	files := map[string]string{
		"device":    "0x" + strings.Split(simulatedDevices[index-1].pciID, ":")[1],
		"numa_node": fmt.Sprintf("%d", numaNode),
		"vendor":    "0x" + strings.Split(simulatedDevices[index-1].pciID, ":")[0],
	}

	// Loop over the files map and create each file with the corresponding content.
	for name, content := range files {
		filePath := filepath.Join(dir, name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to create file %s: %v", filePath, err)
		}
	}
	return nil
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
	err := filepath.Walk(config.pciBasePath, func(path string, info os.FileInfo, err error) error {
		log.Println(config.pciBasePath, info.Name())
		if err != nil {
			return fmt.Errorf("error accessing file path %q", path)
		}
		if info.IsDir() {
			log.Println("Not a device, continuing")
			return nil
		}
		// Retrieve vendor for the device
		vendorID, err := readIDFromFile(config.pciBasePath, info.Name(), "vendor")
		if err != nil {
			return fmt.Errorf("get vendor: %w", err)
		}

		// Habana vendor id is "1da3".
		if vendorID != "1da3" {
			return nil
		}

		deviceID, err := readIDFromFile(config.pciBasePath, info.Name(), "device")
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
	// Simulate returning the number of devices
	return config.DeviceCount, errorString(HLML_SUCCESS)
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
	// We return the Minor number divided by 2 due to the way numbers are generated in the real implementation
	return d.Minor >> 1, nil
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
	// Split the vendor and device parts
	vendor, device := strings.Split(d.pciID, ":")[0], strings.Split(d.pciID, ":")[1]

	// Combine the parts into a single hexadecimal string and convert it to a number
	combinedHex := "0x" + vendor + device
	result, err := strconv.ParseUint(combinedHex, 0, 64)
	if err != nil {
		log.Fatalf("Failed to parse combined hex string: %v\n", err)
		return 0, err
	}

	return uint(result), nil
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

	b, err := os.ReadFile(fmt.Sprintf(config.pciBasePath+"/%s/numa_node", strings.ToLower(busID)))
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
