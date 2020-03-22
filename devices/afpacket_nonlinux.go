// +build !linux

package devices

import (
	"fmt"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// AFPacketHandle is an implementation of a gopacket PacketSource.
type AFPacketHandle struct{}

func newAFPacketHandle(device string, frameSize int, blockSize int, blockCount int, timeout time.Duration) (*AFPacketHandle, error) {
	return nil, fmt.Errorf("AFPacket handles are only available on Linux")
}

// ReadPacketData is an implementation of a gopacket PacketSource's ReadPacketData method.
func (h *AFPacketHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return data, ci, fmt.Errorf("AFPacket handles are only available on Linux")
}

// SetBPFFilter is an implementation of a gopacket PacketSource's SetBPFFilter method.
func (h *AFPacketHandle) SetBPFFilter(filter string, frameSize int) (_ error) {
	return fmt.Errorf("AFPacket handles are only available on Linux")
}

// LinkType is an implementation of a gopacket PacketSource's LinkType method.
func (h *AFPacketHandle) LinkType() layers.LinkType {
	return layers.LinkTypeEthernet
}

// Close is an implementation of a gopacket PacketSource's Close method.
func (h *AFPacketHandle) Close() {}
