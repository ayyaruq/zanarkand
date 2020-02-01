package devices

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"

	"github.com/google/gopacket/pcap"
)

var deviceAnySupported = runtime.GOOS == "linux"

// ListDeviceNames returns a list of available network adapters. The printDescription
// parameter will include the adapter name and printIP will include the IP assigned to it.
func ListDeviceNames(printDescription bool, printIP bool) ([]string, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, err
	}

	list := []string{}
	for _, dev := range devices {
		d := dev.Name

		if printDescription {
			desc := "No description available"
			if len(dev.Description) > 0 {
				desc = dev.Description
			}
			d += fmt.Sprintf(": %s", desc)
		}

		if printIP {
			addresses := "No assigned IP address"
			if len(dev.Addresses) > 0 {
				addresses = ""
				for i, address := range []pcap.InterfaceAddress(dev.Addresses) {
					if i > 0 {
						addresses += " "
					}

					addresses += fmt.Sprintf("%s", address.IP.String())
				}
				d += fmt.Sprintf(" [%s]", addresses)
			}
		}

		list = append(list, d)
	}

	return list, nil
}

// FindDeviceByName returns the device with the provided name.
func FindDeviceByName(name string) (string, error) {
	if name == "" {
		if deviceAnySupported {
			return "any", nil
		}

		return "", errors.New("No device name given")
	}

	device := ""

	if index, err := strconv.Atoi(name); err == nil {
		devices, err := ListDeviceNames(false, false)
		if err != nil {
			return "", fmt.Errorf("Error building device list: %w", err)
		}

		if index >= len(devices) {
			return "", fmt.Errorf("Device index %d/%d out of bounds for device list", index, len(devices))
		}

		device = devices[index]
	}

	return device, nil
}
