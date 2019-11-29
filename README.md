# Zanarkand

Zanarkand is a library to read FFXIV network traffic from PCAP, AF_Packet, or PCAP files. It can
additionally handle TCP reassembly and provides an interface for IPC frame decoding.

For Windows users, elevated security privileges may be required, as well as a local firewall exemption.

To use the library, you need to instantiate a Sniffer and then loop NextFrame in it once it starts.
For each Frame, you can then iterate Messages in it. Helper methods are available to filter Segment
and Opcodes from Frames and Messages respectively. The Sniffer can be stopped and restarted at any time.


## Example

```Go
import (
	"flag"
	"fmt"
	"log"

	"github.com/ayyaruq/zanarkand"
)

func main() {
	// Open flags for debugging if wanted (-assembly_debug_log)
	flag.Parse()

	// Setup the Sniffer
	sniffer, err := zanarkand.NewSniffer("", "en0")
	if err != nil {
		log.Fatal(err)
	}

	// Start the Sniffer goroutine
	go sniffer.Start()

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


## Developing

To start, install Go 1.13 or later. For ease of error handling, the Go 1.13 error wrapping features are used and so this
is the minimum supported version.

Once you have a Go environment setup, install dependencies with `go mod download`.

Zanarkand follows the normal `go fmt` for style. All methods and types should be at least somewhat documented,
beyond that develop as you will as there's no specific expectations. Changes are best submited as pull-requests in GitHub.

Regarding versioning, at this point it's probably overkill, as opcodes and types should be externalised and so there's no
real need to have explicit versions on Zanarkand itself.


## TODO
- [ ] examples
- [ ] tests
- [ ] support fragmented Frames (when a Message spans 2 Frames)
- [ ] other Segment types (currently only IPC seg 3 is implemented)
