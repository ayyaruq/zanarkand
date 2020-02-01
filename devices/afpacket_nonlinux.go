// +build !linux

package devices

import (
	"fmt"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type afpacketHandle struct{}

func newAFPacketHandle(device string, snaplen int, blockSize int, blockCount int, timeout time.Duration) (*afpacketHandle, error) {
	return nil, fmt.Errorf("AFPacket handles are only available on Linux")
}

func (h *afpacketHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return data, ci, fmt.Errorf("AFPacket handles are only available on Linux")
}

func (h *afpacketHandle) SetBPFFilter(filter string) (_ error) {
	return fmt.Errorf("AFPacket handles are only available on Linux")
}

func (h *afpacketHandle) LinkType() layers.LinkType {
	return layers.LinkTypeEthernet
}

func (h *afpacketHandle) Close() {}
