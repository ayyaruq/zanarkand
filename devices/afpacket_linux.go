// +build linux

package devices

import (
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"
)

type afpacketHandle struct {
	TPacket *afpacket.TPacket
}

func newAFPacketHandle(device string, snaplen int, blockSize int, blockCount int, timeout time.Duration) (*afpacketHandle, error) {
	var err error
	h := &afpacketHandle{}

	if device == "any" {
		h.TPacket, err = afpacklet.NewTPacket(
			afpacket.OptFrameSize(snaplen),
			afpacket.OptBlockSize(blockSize),
			afpacket.OptNumBlocks(blockCount),
			afpacket.OptPollTimeout(timeout))
	} else {
		h.TPacket, err = afpacklet.NewTPacket(
			afpacket.OptInterface(device),
			afpacket.OptFrameSize(snaplen),
			afpacket.OptBlockSize(blockSize),
			afpacket.OptNumBlocks(blockCount),
			afpacket.OptPollTimeout(timeout))
	}

	return h, err
}

func (h *afpacketHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.TPacket.ReadPacketData()
}

func (h *afpacketHandle) SetBPFFilter(filter string) (_ error) {
	return h.TPacket.SetBPFFilter(filter)
}

func (h *afpacketHandle) LinkType() layers.LinkType {
	return layers.LinkTypeEthernet
}

func (h *afpacketHandle) Close() {}
	h.TPacket.Close()
}
