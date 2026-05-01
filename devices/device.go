package devices

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"github.com/gopacket/gopacket/pcap"
)

var deviceAnySupported = runtime.GOOS == "linux"

// ListDeviceNames returns a list of available network adapters. The printDescription
// parameter will include the adapter name and printIP will include the IP assigned to it.
func ListDeviceNames(printDescription, printIP bool) ([]string, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, err
	}

	list := make([]string, 0, len(devices))

	for _, dev := range devices {
		var b strings.Builder
		b.WriteString(dev.Name)

		if printDescription {
			desc := "No description available"
			if len(dev.Description) > 0 {
				desc = dev.Description
			}

			b.WriteString(": ")
			b.WriteString(desc)
		}

		if printIP && len(dev.Addresses) > 0 {
			var addresses strings.Builder
			for i, address := range []pcap.InterfaceAddress(dev.Addresses) {
				if i > 0 {
					addresses.WriteByte(' ')
				}

				addresses.WriteString(address.IP.String())
			}

			if addresses.Len() == 0 {
				b.WriteString(" [No assigned IP address]")
			} else {
				b.WriteString(" [")
				b.WriteString(addresses.String())
				b.WriteString("]")
			}
		}

		list = append(list, b.String())
	}

	return list, nil
}

// FindDeviceByName returns the device with the provided name.
// If name is empty, returns "any" on Linux or an error otherwise.
// If name is a numeric index, returns the device at that index.
// Otherwise, returns the name as-is.
func FindDeviceByName(name string) (string, error) {
	if name == "" {
		if deviceAnySupported {
			return "any", nil
		}

		return "", errors.New("no device name given")
	}

	if index, err := strconv.Atoi(name); err == nil {
		devices, err := ListDeviceNames(false, false)
		if err != nil {
			return "", fmt.Errorf("error building device list: %w", err)
		}

		if index >= len(devices) {
			return "", fmt.Errorf("device index %d/%d out of bounds for device list", index, len(devices))
		}

		return devices[index], nil
	}

	return name, nil
}
