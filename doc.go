/*
Package zanarkand is an FFXIV network packet capture and reassembly library.

It reassembles TCP streams carrying FFXIV IPC traffic, decompresses ZLIB-encoded
frame bodies, and dispatches decoded messages to subscribers via channels or
callbacks. Capture sources include live interfaces (pcap, afpacket, pfring) and
offline pcap files.

# Wire format

TCP streams are reassembled into Frames, each containing one or more Messages.

Frame header (40 bytes):

	┌──────┬──────┬──────┬──────┬──────┬──────┬──────┬──────┐
	│  0   │  8   │  16  │  24  │  28  │  30  │  32  │  33  │  offset
	├──────┼──────┼──────┼──────┼──────┼──────┼──────┼──────┤
	│ Magic│ ???  │ Time │Length│Connec│Count │  ?   │Comp. │
	│ (u64)│      │ (u64)│ (u32)│(u16) │(u16) │(byte)│(byte)│
	├──────┴──────┴──────┴──────┴──────┴──────┴──────┼──────┤
	│               ??? (6 bytes)                    │  40  │
	├────────────────────────────────────────────────┼──────┤
	│          Body (Length bytes, optional ZLIB)    │      │
	└────────────────────────────────────────────────┴──────┘

GenericHeader — every Message starts with this 16-byte prefix:

	┌──────┬──────┬──────┬──────┐
	│  0   │  4   │  8   │  12  │  offset
	├──────┼──────┼──────┼──────┤
	│Length│Source│Target│ Seg  │
	│(u32) │(u32) │(u32) │(u16) │
	└──────┴──────┴──────┴──────┘

GameEvent message layout (segment 3):

	┌──────────┬──────┬──────┬──────┬──────┬──────┬──────┐
	│   0      │  16  │  18  │  20  │  22  │  24  │  32  │  offset
	├──────────┼──────┼──────┼──────┼──────┼──────┼──────┤
	│ Generic  │0x1400│Opcode│  ?   │Server│ Time │  ?   │
	│ Header   │(u16) │(u16) │(u16) │(u16) │(u32) │(u32) │
	├──────────┴──────┴──────┴──────┴──────┴──────┼──────┤
	│              Body (opcode-specific)         │      │
	└─────────────────────────────────────────────┴──────┘

Keepalive message layout (segments 7/8):

	┌──────────┬──────┬──────┐
	│   0      │  16  │  20  │  offset
	├──────────┼──────┼──────┤
	│ Generic  │  ID  │ Time │
	│ Header   │(u32) │(u32) │
	└──────────┴──────┴──────┘

# Quick start

Create a Sniffer, attach a subscriber, and process frames:

	sniffer, err := zanarkand.NewSniffer("pcap", "eth0")
	if err != nil {
		log.Fatal(err)
	}

	sub := zanarkand.NewGameEventSubscriber()
	errCh := make(chan error, 1)

	go func() {
		errCh <- sub.Subscribe(context.Background(), sniffer)
	}()

	for msg := range sub.IngressEvents {
		fmt.Printf("opcode 0x%X from actor %d\n", msg.Opcode, msg.SourceActor)
	}

Use a callback-style subscriber for lower overhead:

	handler := zanarkand.NewGameEventHandler(func(msg *zanarkand.GameEventMessage, dir zanarkand.FlowDirection) {
		fmt.Printf("opcode 0x%X direction=%d\n", msg.Opcode, dir)
	})

	go handler.Subscribe(context.Background(), sniffer)

# Capture modes

	newSniffer(mode, source) accepts:

	  "pcap"    — live capture via libpcap
	  "file"    — read from a pcap file
	  "afpacket"— Linux AF_PACKET (Linux only)
	  "pfring"  — ntop PF_RING (Linux only, requires C headers)

# Sniffer lifecycle

Sniffers are context-aware:

	sniffer, _ := zanarkand.NewSniffer("pcap", "eth0")
	ctx, cancel := context.WithCancel(context.Background())

	go sniffer.Start(ctx) // blocks until ctx cancelled or file exhausted

	// Consume frames with NextFrame or ProcessFrames
	frame, err := sniffer.NextFrame()

	// Graceful stop
	sniffer.Stop() // or cancel()

SnifferState tracks the lifecycle: SnifferStopped → SnifferRunning → SnifferFinished (file mode).

# Configuration

Use functional options to tune buffer sizes:

	sniffer, err := zanarkand.NewSniffer("pcap", "eth0",
		zanarkand.WithDataBufferSize(500),
		zanarkand.WithErrorBufferSize(10),
	)

For GameEvent subscribers, filter by opcode:

	sub := zanarkand.NewGameEventSubscriber(
		zanarkand.WithOpcodes(0x031F, 0x0232), // only status effects and actor cast
	)

# Subscriber types

All subscribers implement the Subscriber interface:

	type Subscriber interface {
		Subscribe(ctx context.Context, s *Sniffer) error
		Close(s *Sniffer)
	}

Channel-based (push model):

  - GameEventSubscriber — separate IngressEvents / EgressEvents channels
  - KeepaliveSubscriber — single Events channel

Callback-based (lower overhead, no channel coordination):

  - GameEventHandler — calls GameEventCallback(msg, direction) per message
  - KeepaliveHandler — calls KeepaliveCallback(msg) per message

Callback handlers reuse a single message allocation across calls via Reset()
methods, avoiding per-message heap allocations. The message pointer passed to
the callback is only valid for the duration of the call; copy any data that
must outlive the callback.

Subscribers auto-start the Sniffer if it is not already running.

# Direction inference

Frame.Direction() infers ingress/egress by checking whether the source or
destination IP falls within private address ranges (RFC 1918, loopback,
link-local). Returns FrameIngress (1), FrameEgress (2), or 0 for undetermined.

# Error handling

The package defines four typed errors, all implementing Unwrap() for use with
errors.Is and errors.As:

	ErrNotEnoughData    — payload shorter than declared length
	ErrDecodingFailure  — a specific message could not be decoded
	ErrUnknownInput     — unrecognised capture mode
	ErrReassemblyError  — TCP stream reassembly problem

Reassembly errors are reported on a buffered channel accessible via
Sniffer.Errors(). Errors are dropped silently when the channel is full.

# Debugging and profiling

Pass -assembly_debug_log to your binary for verbose per-packet assembly logging
(gopacket built-in flag).

StartTrace / StopTrace use runtime/trace for profiling channel buffer exhaustion
or frame decode bottlenecks:

	zanarkand.StartTrace(os.Create("trace.out"))
	defer zanarkand.StopTrace()

# Platform support

pcap and file modes work on all platforms. afpacket and pfring are Linux-only
and use build-tagged files in devices/. Non-Linux platforms get stub
implementations that return errors.
*/
package zanarkand
