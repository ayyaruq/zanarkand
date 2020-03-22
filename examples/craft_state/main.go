package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ayyaruq/zanarkand"
)

const OpcodeEventPlay32 = 0x03AF

func main() {
	os.Exit(fakeMain())
}

func fakeMain() int {
	// Setup program control
	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)

	// Load inputs
	var inet = flag.String("i", "en0", "The network interface to capture from")

	flag.Parse()

	// Setup the Sniffer
	sniffer, err := zanarkand.NewSniffer("pcap", *inet)
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
	log.Println("Starting sniffer on interface", *inet)
	go subscriber.Subscribe(sniffer)

	for {
		select {
		case inbound := <-subscriber.IngressEvents:
			if inbound.Opcode == OpcodeEventPlay32 {
				event := new(EventPlay32)
				event.UnmarshalBytes(inbound.Body)
				if (event.EventID == EventIDs["CraftState"]) {
					craftState, ok := event.Data.(CraftState)
					if ok {
						log.Println(json.Marshal(craftState))
					} else {
						log.Println("Unable to validate Event type")
					}
				}
			}

		case <-subscriber.EgressEvents:
			continue

		case sig := <-gracefulStop:
			log.Printf("Received %v signal", sig)
			return 0
		}
	}
}
