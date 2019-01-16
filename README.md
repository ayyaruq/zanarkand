# Zanarkand

Zanarkand is a library to read FFXIV network traffic from PCAP, AF_Packet, or PCAP files. It can
additionally handle TCP reassembly and provides an interface for IPC frame decoding.

For Windows users, elevated security privileges are required, as well as a local firewall exemption.

Two main `io.Readers` are provided. `ReadFrame` will provide a full FFXIV frame, and is mostly useful
for debugging. `ReadMessage` will read a full IPC, however it's up to the user to decode the IPC. It's
worth noting than `ReadMessage` will call `ReadFrame` as required, as it builds upon reassembled frames.

Additionally, users can pull a single TCP packet and optionally extract any frames from it using the
`Framer` package. Since this doesn't require TCP reassembly, it's possible to be missing data, however
this can be useful for dealing with retransmit and protocol rubberbanding.


## Example

```Go
import (
	"fmt"
	"io"

	"github.com/ayyaruq/ayct"
)

func main() {
	s := ayct.Sniffer{ server: 'Yojimbo' }
	f := ayct.Framer{ sniffer: s }

	defer s.Close()

	s.Start()

	// Print a frame metadata
	frame := f.ReadFrame()
	fmt.Println(frame.ToString())

	// Print a message header
	message := f.ReadMessage()
	fmt.Println(message.ToString())

	// Use an io.Reader
	// TODO

	s.Stop()
}
```
