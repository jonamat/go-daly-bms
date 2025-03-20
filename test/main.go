package main

import (
	"fmt"
	"log"

	dalybms "github.com/jonamat/go-daly-bms/pkg/bms"
)

func main() {
	bms := dalybms.NewDalyBMS(3, 4) // 3 retries, address=4 for RS485/USB
	if err := bms.Connect("/dev/ttyUSB0"); err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer bms.Disconnect()

	status, err := bms.GetStatus()
	if err != nil {
		log.Printf("Error reading status: %v", err)
	} else {
		fmt.Printf("Status: %+v\n", status)
	}

	socData, err := bms.GetSOC()
	if err != nil {
		log.Printf("Error reading SOC: %v", err)
	} else {
		fmt.Printf("SOC Data: %+v\n", socData)
	}
}
