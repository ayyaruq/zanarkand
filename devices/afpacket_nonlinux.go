// +build !linux

package devices

import (
	"fmt"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type AFPacketHandle struct{}

func newAFPacketHandle(device string, frameSize int, blockSize int, blockCount int, timeout time.Duration) (*AFPacketHandle, error) {
	return nil, fmt.Errorf("AFPacket handles are only available on Linux")
}

func (h *AFPacketHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return data, ci, fmt.Errorf("AFPacket handles are only available on Linux")
}

func (h *AFPacketHandle) SetBPFFilter(filter string, frameSize int) (_ error) {
	return fmt.Errorf("AFPacket handles are only available on Linux")
}

func (h *AFPacketHandle) LinkType() layers.LinkType {
	return layers.LinkTypeEthernet
}

func (h *AFPacketHandle) Close() {}
