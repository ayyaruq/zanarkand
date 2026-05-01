//go:build !linux

package devices

import (
	"fmt"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

const nolinux = "AF_PACKET handles are only available on Linux"

// AFPacketHandle is an implementation of a gopacket PacketSource.
type AFPacketHandle struct{}

func newAFPacketHandle(device string, frameSize, blockSize, blockCount int, timeout time.Duration) (*AFPacketHandle, error) {
	return nil, fmt.Errorf(nolinux)
}

// ReadPacketData is an implementation of a gopacket PacketSource's ReadPacketData method.
func (h *AFPacketHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return data, ci, fmt.Errorf(nolinux)
}

// SetBPFFilter is an implementation of a gopacket PacketSource's SetBPFFilter method.
func (h *AFPacketHandle) SetBPFFilter(filter string, frameSize int) (_ error) {
	return fmt.Errorf(nolinux)
}

// LinkType is an implementation of a gopacket PacketSource's LinkType method.
func (h *AFPacketHandle) LinkType() layers.LinkType {
	return layers.LinkTypeEthernet
}

// Close is an implementation of a gopacket PacketSource's Close method.
func (h *AFPacketHandle) Close() {}
