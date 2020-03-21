package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ayyaruq/zanarkand"
)

func main() {
	// Setup program control
	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)

	// Load inputs
	var mode = flag.String("m", "pcap", "The sniffer source mode")
	var inet = flag.String("i", "en0", "The network interface to capture from")
	var file = flag.String("f", "", "The file path to capture from")

	flag.Parse()

	// Setup the Sniffer
	var src string
	switch *mode {
	case "file":
		src = *file
	case "pcap":
	default:
		src = *inet
	}

	sniffer, err := zanarkand.NewSniffer(*mode, src)
	if err != nil {
		log.Fatal(err)
	}

	// Close when we're done
	defer func(sniffer *zanarkand.Sniffer) {
		if sniffer.Active {
			sniffer.Stop()
			fmt.Println("stopped active sniifer")
		} else {
			fmt.Println("no active sniffer")
		}
	}(sniffer)

	// Create our message receiver channel
	subscriber := zanarkand.NewGameEventSubscriber()

	// Don't block the Sniffer
	go subscriber.Subscribe(sniffer)

	for {
		select {
		case inbound := <-subscriber.IngressEvents:
			fmt.Printf("Received: %s\n", inbound.String())

		case outbound := <-subscriber.EgressEvents:
			fmt.Printf("Sent: %s\n", outbound.String())

		case sig := <-gracefulStop:
			fmt.Printf("Received %v signal\n", sig)
			os.Exit(0) // this is bad and should be run in a wrapper function since it breaks defers but
		}
	}
}
