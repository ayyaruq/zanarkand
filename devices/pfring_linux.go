//go:build linux

package devices

import (
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pfring"
)

// PFRingHandle is an implementation of a gopacket PacketSource.
type PFRingHandle struct {
	Ring *pfring.Ring
}

func newPFRingHandle(device string, snaplen uint32, timeout time.Duration) (*PFRingHandle, error) {
	ring, err := pfring.NewRing(device, snaplen, pfring.FlagPromisc)
	if err != nil {
		return nil, err
	}

	if err := ring.SetPollDuration(uint(timeout.Milliseconds())); err != nil {
		ring.Close()
		return nil, err
	}

	if err := ring.SetSocketMode(pfring.ReadOnly); err != nil {
		ring.Close()
		return nil, err
	}

	if err := ring.Enable(); err != nil {
		ring.Close()
		return nil, err
	}

	return &PFRingHandle{Ring: ring}, nil
}

// ReadPacketData is an implementation of a gopacket PacketSource's ReadPacketData method.
func (h *PFRingHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.Ring.ReadPacketData()
}

// SetBPFFilter is an implementation of a gopacket PacketSource's SetBPFFilter method.
func (h *PFRingHandle) SetBPFFilter(filter string) error {
	return h.Ring.SetBPFFilter(filter)
}

// LinkType is an implementation of a gopacket PacketSource's LinkType method.
func (h *PFRingHandle) LinkType() layers.LinkType {
	return layers.LinkTypeEthernet
}

// Close is an implementation of a gopacket PacketSource's Close method.
func (h *PFRingHandle) Close() {
	h.Ring.Close()
}
