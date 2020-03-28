# Zanarkand

[![Build Status](https://img.shields.io/github/workflow/status/ayyaruq/zanarkand/Go%20Test)](https://github.com/ayyaruq/zanarkand/actions)
[![Dependencies](https://img.shields.io/librariesio/github/ayyaruq/zanarkand)](https://libraries.io/github/ayyaruq/zanarkand)
[![Code Quality](https://goreportcard.com/badge/github.com/ayyaruq/zanarkand)](https://goreportcard.com/report/github.com/ayyaruq/zanarkand)
[![GitHub Issues](https://img.shields.io/github/issues/ayyaruq/zanarkand.svg)](https://github.com/ayyaruq/zanarkand/issues)
[![GitHub Pull Requests](https://img.shields.io/github/issues-pr/ayyaruq/zanarkand.svg)](https://github.com/ayyaruq/zanarkand/pulls)
[![GitHub License](https://img.shields.io/github/license/ayyaruq/zanarkand.svg)](https://github.com/ayyaruq/zanarkand/blob/master/LICENSE)
![Programming Language](https://img.shields.io/github/languages/top/ayyaruq/zanarkand)
[![Discord](https://img.shields.io/discord/479945159203880960?color=7289da&label=discord&logo=discordo)](https://discord.gg/fwUwjB5)

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
	sniffer, err := zanarkand.NewSniffer("pcap", "en0")
	if err != nil {
		log.Fatal(err)
	}

	// Create a channel to receive Messages on
	subscriber := zanarkand.NewGameEventSubscriber()

	// Don't block the Sniffer, but capture errors
	go func() {
		err := subscriber.Subscribe(sniffer)
		if err != nil {
			log.Fatal(err)
		}
	}()

// Capture the first 10 Messages sent from the server
// This ignores Messages sent by the client to the server
	for i := 0, i < 10; i++ {
		message := <-subscriber.IngressEvents
		fmt.Println(message.String())
	}

	// Stop the sniffer
	subscriber.Close()
}
```


## Developing

To start, install Go 1.13 or later. For ease of error handling, the Go 1.13 error wrapping features are used and so this
is the minimum supported version.

To add to your project, simple `go mod init` if you don't already have a `go.mod` file, and then
`go get -u github.com/ayyaruq/zanarkand`.


## Contributing

Once you have a Go environment setup, install dependencies with `make deps`.

Zanarkand follows the normal `go fmt` for style. All methods and types should be at least somewhat documented,
beyond that develop as you will as there's no specific expectations. Changes are best submited as pull-requests in GitHub.

Regarding versioning, at this point it's probably overkill, as opcodes and types are externalised and so there's no
real need to have explicit versions on Zanarkand itself.


## TODO
- [ ] [better error wrapping](https://github.com/ayyaruq/zanarkand/issues/4)
- [ ] [winsock capture from a PID via the TCP table, requires iphlpapi](https://github.com/ayyaruq/zanarkand/issues/3)
- [ ] support fragmented Frames (when a Message spans 2 Frames)
