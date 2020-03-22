// +build linux

package devices

import (
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"golang.org/x/net/bpf"
)

// AFPacketHandle is an implementation of a gopacket PacketSource.
type AFPacketHandle struct {
	TPacket *afpacket.TPacket
}

func newAFPacketHandle(device string, frameSize int, blockSize int, blockCount int, timeout time.Duration) (*AFPacketHandle, error) {
	var err error
	h := &AFPacketHandle{}

	if device == "any" {
		h.TPacket, err = afpacket.NewTPacket(
			afpacket.OptFrameSize(frameSize),
			afpacket.OptBlockSize(blockSize),
			afpacket.OptNumBlocks(blockCount),
			afpacket.OptPollTimeout(timeout))
	} else {
		h.TPacket, err = afpacket.NewTPacket(
			afpacket.OptInterface(device),
			afpacket.OptFrameSize(frameSize),
			afpacket.OptBlockSize(blockSize),
			afpacket.OptNumBlocks(blockCount),
			afpacket.OptPollTimeout(timeout))
	}

	return h, err
}

// ReadPacketData is an implementation of a gopacket PacketSource's ReadPacketData method.
func (h *AFPacketHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.TPacket.ReadPacketData()
}

// SetBPFFilter is an implementation of a gopacket PacketSource's SetBPFFilter method.
func (h *AFPacketHandle) SetBPFFilter(filter string, frameSize int) (_ error) {
	pcapBPF, err := pcap.CompileBPFFilter(h.LinkType(), frameSize, filter)
	if err != nil {
		return err
	}

	instructions := []bpf.RawInstruction{}
	for _, ins := range pcapBPF {
		rawins := bpf.RawInstruction{
			Op: ins.Code,
			Jt: ins.Jt,
			Jf: ins.Jf,
			K:  ins.K,
		}
		instructions = append(instructions, rawins)
	}

	return h.TPacket.SetBPF(instructions)
}

// LinkType is an implementation of a gopacket PacketSource's LinkType method.
func (h *AFPacketHandle) LinkType() layers.LinkType {
	return layers.LinkTypeEthernet
}

// Close is an implementation of a gopacket PacketSource's Close method.
func (h *AFPacketHandle) Close() {
	h.TPacket.Close()
}
