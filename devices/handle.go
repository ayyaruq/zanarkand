package devices

import (
	"os"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// DeviceHandle is an implementation of the gopacket PacketDataSource.
// A custom handle is used so that we can stub out different device types
// across platforms.
type DeviceHandle interface {
	gopacket.PacketDataSource

	LinkType() layers.LinkType
	Close()
}

// OpenPcap opens a DeviceHandle for a live PCAP session on a given interface.
func OpenPcap(device string, filter string, timeout time.Duration) (*pcap.Handle, error) {
	h, err := pcap.OpenLive(device, 1600, true, timeout)
	if err != nil {
		return nil, err
	}

	err = h.SetBPFFilter(filter)
	if err != nil {
		h.Close()
		return nil, err
	}

	return h, nil
}

// OpenFile opens a DeviceHandle for an offline PCAP session with a given input file.
func OpenFile(file string, filter string) (*pcap.Handle, error) {
	h, err := pcap.OpenOffline(file)
	if err != nil {
		return nil, err
	}

	err = h.SetBPFFilter(filter)
	if err != nil {
		h.Close()
		return nil, err
	}

	return h, nil
}

// OpenAFPacket opens a DeviceHandle for live capture via AF_Packet on a given interface.
// The buffer size depends on system memory, with frame and block sizes calculated from
// the size of the buffer, system page size, and a default snaplen of 1600. Generally
// multiples of 25MB are a good size, since 1600 bytes is an even multiple when page sizes
// are in powers of 2. By default, 128 blocks can fit in 25MB. AF_Packet is only available
// on Linux systems.
func OpenAFPacket(device string, filter string, bufferSize int, timeout time.Duration) (*AFPacketHandle, error) {
	frameSize, blockSize, blockCount, err := afpacketCalculateBuffers(bufferSize, 1600, os.Getpagesize())
	if err != nil {
		return nil, err
	}

	h, err := newAFPacketHandle(device, frameSize, blockSize, blockCount, timeout)
	if err != nil {
		return nil, err
	}

	err = h.SetBPFFilter(filter, frameSize)
	if err != nil {
		h.Close()
		return nil, err
	}

	return h, nil
}
