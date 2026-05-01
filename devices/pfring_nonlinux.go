//go:build !linux

package devices

import (
	"fmt"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

const nolinux = "PF_RING handles are only available on Linux"

// PFRingHandle is a stub for non-Linux platforms.
type PFRingHandle struct{}

func newPFRingHandle(device string, snaplen uint32, timeout time.Duration) (*PFRingHandle, error) {
	return nil, fmt.Errorf(nolinux)
}

// ReadPacketData is an implementation of a gopacket PacketSource's ReadPacketData method.
func (h *PFRingHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return data, ci, fmt.Errorf(nolinux)
}

// SetBPFFilter is an implementation of a gopacket PacketSource's SetBPFFilter method.
func (h *PFRingHandle) SetBPFFilter(filter string) error {
	return fmt.Errorf(nolinux)
}

// LinkType is an implementation of a gopacket PacketSource's LinkType method.
func (h *PFRingHandle) LinkType() layers.LinkType {
	return layers.LinkTypeEthernet
}

// Close is an implementation of a gopacket PacketSource's Close method.
func (h *PFRingHandle) Close() {}
