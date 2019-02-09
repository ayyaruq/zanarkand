# Zanarkand

Zanarkand is a library to read FFXIV network traffic from PCAP, AF_Packet, or PCAP files. It can
additionally handle TCP reassembly and provides an interface for IPC frame decoding.

For Windows users, elevated security privileges are required, as well as a local firewall exemption.

To use the library, you need to instantiate a Sniffer and then loop NextFrame in it once it starts.
For each Frame, you can then iterate Messages in it. Helper methods are available to filter Segment
and Opcodes from Frames and Messages respectively. The Sniffer can be stopped and restarted at any time.


## Example

```Go
import (
	"fmt"
	"log"

	"github.com/ayyaruq/zanarkand"
)

func main() {
	// Setup the Sniffer
	sniffer, err := zanarkand.NewSniffer("", "en0")
	if err != nil {
		log.Fatal(err)
	}

	// Start the Sniffer
	sniffer.Start()

	for i := 0, i < 10; i++ {
		frame, err := sniffer.NextFrame()
		if err != nil {
			log.Print(err)
		}

		// Print the Message Headers
		for _, message := range frame.Messages {
			fmt.Println(message.Header.ToString())
		}
	}

	sniffer.Stop()
}
```


## TODO
- [ ] examples
- [ ] some interface methods for easier access and extraction
- [ ] tests
- [ ] updated opcode registry
- [ ] type deserialisation?
- [ ] support fragmented Frames (when a Message spans 2 Frames)
- [ ] other Segment types (currently only IPC seg 3 is implemented)
- [ ] io.Reader into user Buffer, but this is a huge pain
