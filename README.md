# Zanarkand

[![Build Status](https://img.shields.io/github/actions/workflow/status/ayyaruq/zanarkand/test.yml?branch=master)](https://github.com/ayyaruq/zanarkand/actions)
[![Code Quality](https://goreportcard.com/badge/github.com/ayyaruq/zanarkand)](https://goreportcard.com/report/github.com/ayyaruq/zanarkand)
[![GitHub Issues](https://img.shields.io/github/issues/ayyaruq/zanarkand.svg)](https://github.com/ayyaruq/zanarkand/issues)
[![GitHub Pull Requests](https://img.shields.io/github/issues-pr/ayyaruq/zanarkand.svg)](https://github.com/ayyaruq/zanarkand/pulls)
[![GitHub License](https://img.shields.io/github/license/ayyaruq/zanarkand.svg)](https://github.com/ayyaruq/zanarkand/blob/master/LICENSE)
[![Discord](https://img.shields.io/discord/479945159203880960?color=7289da&label=discord&logo=discordo)](https://discord.gg/fwUwjB5)

Zanarkand is a library to read FFXIV network traffic from PCAP, AF_Packet, PF_RING, or PCAP files. It can
additionally handle TCP reassembly and provides an interface for IPC frame decoding.

For Windows users, elevated security privileges may be required, as well as a local firewall exemption.

To use the library, instantiate a Sniffer and call `Start(ctx)` with a context. The Sniffer blocks until
`Stop()` is called or the context is cancelled. Frames are consumed via `NextFrame()`, which returns when
a frame is available or the Sniffer stops. Helper subscribers are available to filter Segment types and
deliver decoded messages on channels. The Sniffer can be stopped at any time via `Stop()` or context cancellation.


## Example

```Go
import (
	"context"
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
		err := subscriber.Subscribe(context.Background(), sniffer)
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Capture the first 10 Messages sent from the server
	// This ignores Messages sent by the client to the server
	for i := 0; i < 10; i++ {
		message := <-subscriber.IngressEvents
		fmt.Println(message.String())
	}

	// Stop the sniffer
	subscriber.Close(sniffer)
}
```


## Debugging

### Verbose TCP assembly logging

gopacket's tcpassembly package has a hidden flag `-assembly_debug_log` that logs at least one line per packet. Pass it to your binary:

```bash
./myparser -assembly_debug_log -i en0
```

**Important:** If your tool calls `flag.Parse()`, this flag will be recognized automatically. If you set flags programmatically (e.g. `flag.Set`), call `flag.Parse()` before starting the sniffer, or set the flag after parsing.

### Profiling with runtime/trace

If you experience performance issues (e.g., channel buffer exhaustion under high packet volume),
use `runtime/trace` to profile:

```go
f, err := os.Create("trace.out")
if err != nil {
	log.Fatal(err)
}
defer f.Close()

if err := zanarkand.StartTrace(f); err != nil {
	log.Fatal(err)
}
defer zanarkand.StopTrace()

// ... run your capture ...
```

View the trace with: `go tool trace trace.out`

### Reassembler errors

The Sniffer exposes an error channel for TCP reassembly failures:

```go
go func() {
	for err := range sniffer.Errors() {
		log.Printf("reassembler error: %v", err)
	}
}()
```


## Developing

To start, install Go 1.24 or later. Error types implement `Unwrap()` for Go 1.13+ error wrapping.

To add to your project, simple `go mod init` if you don't already have a `go.mod` file, and then
`go get -u github.com/ayyaruq/zanarkand`.


## Contributing

Once you have a Go environment setup, install dependencies with `make deps`.

Zanarkand follows the normal `gofmt` for style. All methods and types should be at least somewhat documented,
beyond that develop as you will as there's no specific expectations. Changes are best submited as pull-requests in GitHub.

Regarding versioning, at this point it's probably overkill, as opcodes and types are externalised and so there's no
real need to have explicit versions on Zanarkand itself.


## TODO
- [ ] [better error wrapping](https://github.com/ayyaruq/zanarkand/issues/4)
- [ ] [winsock capture from a PID via the TCP table, requires iphlpapi](https://github.com/ayyaruq/zanarkand/issues/3)
- [ ] support fragmented Frames (when a Message spans 2 Frames)
