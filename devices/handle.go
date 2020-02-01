package devices

import (
	"os"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type DeviceHandle interface {
	gopacket.PacketDataSource

	LinkType() layers.LinkType
	Close()
}

func OpenPcap(device string, filter string, timeout time.Duration) (DeviceHandle, error) {
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

func OpenFile(file string, filter string) (DeviceHandle, error) {
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

func OpenAFPacket(device string, filter string, bufferSize int, timeout time.Duration) (DeviceHandle, error) {
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
