package devices

import "errors"

// Calculate the size of the mmap buffers used for an AFPacket handle.
// The block size and block count should add up to as close as possible
// to the target allocation size. Block size must be divisible by both
// the frame and page size however. TargetSize is in MB.
func afpacketCalculateBuffers(targetSize int, snaplen int, pageSize int) (frameSize, blockSize, blockCount int, err error) {
	if snaplen < pageSize {
		frameSize = pageSize / (pageSize / snaplen)
	} else {
		frameSize = (snaplen/pageSize + 1) * pageSize
	}

	blockSize = frameSize * 128 // Default in gopacket
	blockCount = (targetSize * 1024 * 1024) / blockSize

	if blockCount == 0 {
		return 0, 0, 0, errors.New("Buffer size too small")
	}

	return frameSize, blockSize, blockCount, nil
}
