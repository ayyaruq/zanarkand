package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ayyaruq/zanarkand"
)

func main() {
	os.Exit(fakeMain())
}

func fakeMain() int {
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
		return 1
	}

	// Create our message receiver channel
	subscriber := zanarkand.NewGameEventSubscriber()

	// Close when we're done
	defer func(sniffer *zanarkand.Sniffer) {
		if sniffer.Active {
			subscriber.Close(sniffer)
			log.Println("Stopped active snifer")
		}
	}(sniffer)

	// Don't block the Sniffer
	log.Println("Starting sniffer from source", src)
	go subscriber.Subscribe(sniffer)

	for {
		select {
		case inbound := <-subscriber.IngressEvents:
			log.Printf("Received: %s", inbound.String())

		case outbound := <-subscriber.EgressEvents:
			log.Printf("Sent: %s", outbound.String())

		case sig := <-gracefulStop:
			log.Printf("Received %v signal", sig)
			return 0
		}
	}
}
