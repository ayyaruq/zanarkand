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
	var inet = flag.String("i", "", "The network interface to capture from")
	var file = flag.String("f", "", "The file path to capture from")

	flag.Parse()

	// Setup the Sniffer
	sniffer, err := zanarkand.NewSniffer(*file, *inet)
	if err != nil {
		log.Fatal(err)
	}

	// Close when we're done
	defer func() {
		if sniffer.Active() {
			sniffer.Stop()
			fmt.Println("stopped active sniifer")
		} else {
			fmt.Println("no active sniffer")
		}
	}()

	// Create our message receiver channel
	subscriber := zanarkand.NewGameEventSubscriber()

	// Don't block the Sniffer
	go subscriber.Subscribe(sniffer)

	for {
		select {
		case message := <-subscriber.Events:
			fmt.Printf("%v\n", message.String())

		case sig := <-gracefulStop:
			fmt.Printf("Received %v signal\n", sig)
			os.Exit(0)
		}
	}
}
