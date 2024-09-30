// hlml_fake.go
//go:build fake
// +build fake

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

// Device struct for fake devices.
type Device struct {
	serialNumber string
	uuid         string
	pciID        string
	pciBusID     string
	Minor        uint
	Module       uint
}

// EventSet is a fake implementation of the HLML event set.
type EventSet struct{}

// Event is a fake implementation of the HLML event.
type Event struct {
	Serial string
	Etype  uint64
}

type HLMLReturn int

// FakeHlml simulates the HLML library behavior.
type FakeHlml struct{}

// FakeDeviceConfig is a struct that used to parse the YAML configuration for the fake devices.
type FakeDeviceConfig struct {
	Path          string `yaml:"Path"`
	PciID         string `yaml:"PciID"`
	HLDevice      string `yaml:"HLDevice"`
	pciBasePath   string
	devBasePath   string
	DeviceCount   uint    `yaml:"DeviceCount"`
	NumaNodes     uint    `yaml:"NumaNodes"`
	UnhealthyFreq float64 `yaml:"UnhealthyFreq"`
	TimeoutFreq   float64 `yaml:"TimeoutFreq"`
}

// HlmlSuccess defines the success return code to fake device no errors needed.
const HlmlSuccess HLMLReturn = 0
const HlmlErrorTimeout HLMLReturn = 10

// EventType defines the type of event.
const (
	HlmlEventEccErr      = 1 << 0 // Event about ECC errors
	HlmlEventCriticalErr = 1 << 1 // Event about critical errors that occurred on the device
	HlmlEventClockRate   = 1 << 2 // Event about changes in clock rate
)

var (
	ErrInvalidHlmlErrorCode        = errors.New("invalid HLML error return code")
	ErrParsingConfig               = errors.New("error parsing config file")
	ErrRemoveExistingDirectory     = errors.New("failed to remove existing directory")
	ErrCreateDirectory             = errors.New("failed to create directory")
	ErrCreateDeviceNode            = errors.New("failed to create device node")
	ErrCreateTargetDirectory       = errors.New("failed to create target directory")
	ErrCreateSymlink               = errors.New("failed to create symlink")
	ErrCreateFilesInDirectory      = errors.New("failed to create files in directory")
	ErrCreateFile                  = errors.New("failed to create file")
	ErrAccessFilePath              = errors.New("error accessing file path")
	ErrCouldNotFindDeviceBySerial  = errors.New("could not find device with serial number")
	ErrCouldNotFindDeviceByIndex   = errors.New("could not find device with index")
	ErrNoHabanaDevices             = errors.New("no habana devices on the system")
	ErrSerialNumberUnavailable     = errors.New("SerialNumber not available")
	ErrUUIDUnavailable             = errors.New("UUID not available")
	ErrPCIBusIDUnavailable         = errors.New("PCIBusID not available")
	ErrFailedToRetrieveCPUAffinity = errors.New("failed to retrieve CPU affinity")
	ErrEventTimeout                = errors.New("event timeout")
)

// Global variables holding the simulated devices.
var (
	config                   FakeDeviceConfig
	prefix                   string
	simulatedDevices         map[uint]*Device   // Access devices by index
	simulatedDevicesBySerial map[string]*Device // Access devices by serial number
	// Global map to track registered events by UUID.
	registeredEventsByUUID = make(map[string][]int)
)

// errorString translates the HLML return code into a Go error.
func errorString(ret HLMLReturn) error {
	switch ret {
	case HlmlSuccess:
		return nil
	case HlmlErrorTimeout:
		return fmt.Errorf("hlml: %w", ErrEventTimeout)
	}

	return ErrInvalidHlmlErrorCode
}

// updateConfig updates the global `config` variable with the parsed YAML configuration.
func updateConfig(yamlConfig string) error {
	// Parse the YAML string into the Config struct
	err := yaml.Unmarshal([]byte(yamlConfig), &config)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrParsingConfig, err)
	}

	// Update the global config variables
	config.pciBasePath = config.Path + "/sys/bus/pci/devices"
	config.devBasePath = config.Path + "/dev/accel"
	prefix = config.Path

	return nil
}

// initializeSimulatedDevices initializes the global variable `simulatedDevices` with the specified number of devices.
func initializeSimulatedDevices(config FakeDeviceConfig) {
	simulatedDevices = make(map[uint]*Device)
	simulatedDevicesBySerial = make(map[string]*Device)

	// Create a new random generator instance.
	rng := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
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

// generateRandomSerialNumber creates a string like e.g. `AN45012345` where the last 4 digits are random.
func generateRandomSerialNumber() string {
	const baseprefix = "FK" //YYXXXXXXXX
	// Generate random last eight digits.
	// Create a new random generator instance.
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))     //nolint:gosec
	lastFourDigits := fmt.Sprintf("%08d", rng.Intn(100000000)) // Random number between 00000000 and 99999999

	return baseprefix + lastFourDigits
}

// generateRandomUUID creates a string in the format e.g. `01P0-HL2080A0-15-TNBS72-05-01-02`.
func generateRandomUUID() string {
	/*
		Alphanumeric string made up of table_version, device_ID, FAB#, LOT#, Wafer#, X Coordinate, Y Coordinate. Example: 00P1-HL2000B0-14-P63B83-04-08-10
	*/
	// Define possible suffixes
	lots := []string{"P73B93", "TNBR62", "P53B53", "TNBR72"}

	// Create a new random generator instance
	rng := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	// Select a random lot using the local random generator
	lot := lots[rng.Intn(len(lots))]

	fab := "99"

	deviceID := config.HLDevice

	tableVersion := "01F0"

	// Generate a random w,x,y parts
	yCoord := fmt.Sprintf("%02d", rng.Intn(24))   // Random between 00-50
	wafer := fmt.Sprintf("%02d", rng.Intn(12)+1)  // Random between 00-50
	xCoord := fmt.Sprintf("%02d", rng.Intn(28)+1) // Random between 00-50

	// Construct and return the final UUID string
	return fmt.Sprintf("%s-%s-%s-%s-%s-%s-%s", tableVersion, deviceID, fab, lot, wafer, xCoord, yCoord)
}

// getHlml returns the fake HLML implementation when `realhlml` build tag is not used.
func getHlml() Hlml {
	yamlContent := `
Path: "/tmp/gaudi2"
HLDevice: "HL2080F0"
DeviceCount: 8
NumaNodes: 2
PciID: "1da3:1020"
UnhealthyFreq: 0.1
TimeoutFreq: 0.1
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

// createDeviceNodes creates the device nodes for the specified number of devices.
func createDeviceNodes(path string, count uint) error {
	// Remove the existing directory (if it exists) before creating a new one
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("%w %s: %v", ErrRemoveExistingDirectory, path, err)
	}

	// Create the target directory if it does not exist
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("%w %s: %v", ErrCreateDirectory, path, err)
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

// createDeviceNode creates a device node with the specified path, major, and minor number.
func createDeviceNode(path string, major, minor uint32, mode uint32) error {
	dev := int((major << 8) | minor) // Combine major and minor to create a device ID
	if err := syscall.Mknod(path, mode, dev); err != nil {
		return fmt.Errorf("%w, %s: %v", ErrCreateDeviceNode, path, err)
	}

	return nil
}

// createSymlinkedDirectories creates symlinked directories and the target folders with files.
func createSymlinkedDirectories(path string, count uint, numaNodes uint) error {
	// Remove the existing directory (if it exists) before creating a new one
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("%w %s: %v", ErrRemoveExistingDirectory, path, err)
	}

	// Create the target directory if it does not exist
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("%w %s: %v", ErrCreateDirectory, path, err)
	}

	// Calculate how many devices per NUMA node
	devicesPerNode := count / numaNodes
	if devicesPerNode < 1 {
		devicesPerNode = 1 // Ensure at least one device per NUMA node to avoid division by zero
	}

	for i := uint(1); i <= count; i++ {
		// Get the PCI Bus ID for the current device from the simulatedDevices array
		device := simulatedDevices[i-1]

		// Create the symlink name
		symlinkName := fmt.Sprintf("%s/%s", path, device.pciBusID)

		// Extract the PCI root and the target directory path based on the PCI Bus ID
		pciRoot := device.pciBusID[:9] // Extract "0000:0a" from "0000:0a:1f.1"
		targetDir := fmt.Sprintf("../../../devices/pci%s/%s", pciRoot, device.pciBusID)

		// Create the absolute path of the target directory
		fullTargetPath := filepath.Join(path, targetDir)

		// Create the target directory structure if it doesn't exist
		if err := os.MkdirAll(fullTargetPath, 0755); err != nil {
			return fmt.Errorf("%w, %s: %v", ErrCreateTargetDirectory, fullTargetPath, err)
		}

		// Create the symlink in the path directory pointing to the target directory
		if err := os.Symlink(targetDir, symlinkName); err != nil {
			return fmt.Errorf("%w %s -> %s: %v", ErrCreateSymlink, symlinkName, targetDir, err)
		}

		// Determine the NUMA node for this device
		numaNode := (i - 1) / devicesPerNode

		// Create the files inside the target directory with the corresponding NUMA node value
		if err := createFilesInDirectory(fullTargetPath, i, numaNode); err != nil {
			return fmt.Errorf("%w %s: %v", ErrCreateFilesInDirectory, fullTargetPath, err)
		}
	}

	return nil
}

// createFilesInDirectory creates the specified files in the given directory.
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
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil { //nolint:gosec
			return fmt.Errorf("%w %s: %v", ErrCreateFile, filePath, err)
		}
	}

	return nil
}

// Initialize simulates the initialization of the HLML library.
func (d *FakeHlml) Initialize() error {
	// Simulate a successful initialization
	return errorString(HlmlSuccess)
}

// Shutdown simulates the shutdown of the HLML library in the fake implementation.
func (d *FakeHlml) Shutdown() error {
	// Simulate a successful shutdown
	return errorString(HlmlSuccess)
}

// GetDeviceTypeName simulates the retrieval of the device type name in the fake implementation.
func (d *FakeHlml) GetDeviceTypeName() (string, error) {
	var deviceType string

	err := filepath.Walk(config.pciBasePath, func(path string, info os.FileInfo, err error) error {
		log.Println(config.pciBasePath, info.Name())

		if err != nil {
			return fmt.Errorf("%w %q", ErrAccessFilePath, path)
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

// DeviceCount simulates the retrieval of the number of Habana devices in the system.
func (d *FakeHlml) DeviceCount() (uint, error) {
	// Simulate returning the number of devices
	return config.DeviceCount, errorString(HlmlSuccess)
}

// DeviceHandleBySerial simulates getting a handle to a particular device by serial number.
func (d *FakeHlml) DeviceHandleBySerial(serial string) (*Device, error) {
	// Check if the device with the given serial number exists
	if device, found := simulatedDevicesBySerial[serial]; found {
		return device, nil
	}

	// Return an error if the device is not found
	return nil, ErrCouldNotFindDeviceBySerial
}

// NewEventSet simulates creating a new event set in the fake implementation.
func (d *FakeHlml) NewEventSet() *EventSet {
	// Simulate creating a new event set
	return &EventSet{}
}

// DeleteEventSet simulates deleting an event set in the fake implementation.
func (d *FakeHlml) DeleteEventSet(es *EventSet) {
	// Simulate deleting the event se
}

// RegisterEventForDevice simulates registering an event for a device in the fake implementation.
func (d *FakeHlml) RegisterEventForDevice(es *EventSet, event int, uuid string) error {
	// Add the event to the registeredEventsByUUID map
	// Actual uuid is the serial number of the device
	registeredEventsByUUID[uuid] = append(registeredEventsByUUID[uuid], event)
	return errorString(HlmlSuccess)
}

// WaitForEvent simulates waiting for an event in the fake implementation.
func (d *FakeHlml) WaitForEvent(es *EventSet, timeout int) (*Event, error) {
	// Create a random number generator.
	rng := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec

	// Select a random device index within the range of DeviceCount.
	randomIndex := uint(rng.Intn(int(config.DeviceCount)))
	device, found := simulatedDevices[randomIndex]
	if !found {
		return nil, ErrCouldNotFindDeviceByIndex // Return an error if the device is not found
	}

	// Get the serial number of the randomly selected device.
	serialNumber, err := device.SerialNumber()
	if err != nil {
		return nil, err
	}

	if events, exists := registeredEventsByUUID[serialNumber]; exists {
		// Check if this device has the registered event we want
		if isEventRegistered(events, HlmlEventCriticalErr) {
			// Determine if a timeout should be simulated based on the TimeoutFrequency.
			if rng.Float64() < config.TimeoutFreq {
				time.Sleep(time.Duration(timeout) * time.Millisecond) // Simulate waiting for the event.
				return nil, errorString(HlmlErrorTimeout)
			}
			// Determine if the device should return a critical error event based on the UnhealthyFrequency.
			if rng.Float64() < config.UnhealthyFreq {
				// Return a critical error event with the randomly selected device's serial number.
				e := &Event{Serial: serialNumber, Etype: HlmlEventCriticalErr}
				return e, nil
			}
		}
	}

	// In the default case, we return a fake event indicating a healthy status.
	e := &Event{Serial: serialNumber, Etype: 0}
	return e, nil
}

// isEventRegistered checks if a specific event is in the list of registered events.
func isEventRegistered(events []int, event int) bool {
	for _, registeredEvent := range events {
		if registeredEvent == event {
			return true
		}
	}
	return false
}

// DeviceHandleByIndex simulates getting a handle to a device by its index.
func (d *FakeHlml) DeviceHandleByIndex(index uint) (Device, error) {
	// Check if the device with the given index exists
	if device, found := simulatedDevices[index]; found {
		return *device, nil
	} else {
		// Return an error if the device is not found
		return Device{}, ErrCouldNotFindDeviceByIndex
	}
}

// GetCriticalErrorCode returns a simulated critical error code.
func (d *FakeHlml) HlmlCriticalError() uint64 {
	return 1 << 1
}

// MinorNumber simulates returning the Minor number in the fake implementation.
func (d Device) MinorNumber() (uint, error) {
	// Simulate returning a minor number (hardcoded or configurable in the fake struct)
	// We return the Minor number divided by 2 due to the way numbers are generated in the real implementation
	return d.Minor >> 1, nil
}

// ModuleID simulates returning the ModuleID in the fake implementation.
func (d Device) ModuleID() (uint, error) {
	// Simulate returning a module ID (hardcoded or configurable in the fake struct)
	return d.Module, nil
}

// getDeviceName returns the name of the device based on the device ID.
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
		return "", ErrNoHabanaDevices
	}
}

// checkFamily checks if the device ID belongs to the specified family.
func checkFamily(family []string, id string) bool {
	for _, m := range family {
		if strings.HasSuffix(id, m) {
			return true
		}
	}

	return false
}

// readIDFromFile reads the ID from the specified file.
func readIDFromFile(basePath string, deviceAddress string, property string) (string, error) {
	data, err := os.ReadFile(filepath.Join(basePath, deviceAddress, property))
	if err != nil {
		return "", fmt.Errorf("could not read %s for device %s: %w", property, deviceAddress, err)
	}

	id := strings.Trim(string(data[2:]), "\n")

	return id, nil
}

// PCIID returns the PCI ID of the device.
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

// SerialNumber returns the Serial Number of the device.
func (d *Device) SerialNumber() (string, error) {
	// Return the Serial Number of the device
	if d.serialNumber == "" {
		return "", ErrSerialNumberUnavailable
	}

	return d.serialNumber, nil
}

// UUID returns the UUID of the device.
func (d *Device) UUID() (string, error) {
	// Return the UUID of the device
	if d.uuid == "" {
		return "", ErrUUIDUnavailable
	}

	return d.uuid, nil
}

// PCIBusID returns the PCI Bus ID of the device.
func (d *Device) PCIBusID() (string, error) {
	// Return the PCI Bus ID of the device
	if d.pciBusID == "" {
		return "", ErrPCIBusIDUnavailable
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
		return nil, fmt.Errorf("%w: %v", ErrFailedToRetrieveCPUAffinity, err)
	}

	if node < 0 {
		return nil, nil
	}

	numaNode := uint(node)

	return &numaNode, nil
}
