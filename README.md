# Zanarkand

Zanarkand is a library to read FFXIV network traffic from PCAP, AF_Packet, or PCAP files. It can
additionally handle TCP reassembly and provides an interface for IPC frame decoding.

For Windows users, elevated security privileges are required, as well as a local firewall exemption.

Two main `io.Readers` are provided. `ReadFrame` will provide a full FFXIV frame, and is mostly useful
for debugging. `ReadMessage` will read a full IPC, however it's up to the user to decode the IPC. It's
worth noting than `ReadMessage` will call `ReadFrame` as required, as it builds upon reassembled frames.

`Messages()` will pull individual Messages out of a Frame, while ReadMessage is an `io.Reader` for
ingesting Messages into a buffer. `GetMessage()` does the same thing, but for a single Message instead
of an array.


## Example

```Go
import (
	"bufio"
	"fmt"
	"io"

	"github.com/ayyaruq/zanarkand"
)

func main() {
	s := zanarkand.Sniffer{ server: 'Yojimbo' }
	buf := bufio.NewReader(&s.r)

	defer s.Close()

	s.Run()

	// Print a frame metadata
	frame, err := zanarkand.ReadFrame(buf)
	if err == io.EOF {
		// Read unntil EOF
		return
	} else if err != nil {
		log.Println("Error reading stream", err)
	} else {
		fmt.Println(frame.ToString())
		s.Stop()
	}

	// Print some message headers
	for _, message := range frame.Data.Messages() {
		fmt.Println(message.ToString())
	}

}
```
