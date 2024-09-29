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

// Hlml interface defines methods for interacting with the HLML library (real or fake).
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
