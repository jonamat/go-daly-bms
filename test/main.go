package main

import (
	"fmt"
	"time"

	bms "github.com/jonamat/go-daly-bms"
)

const SAMPLE_INTERVAL = 5
const BMS_PORT = "/dev/ttyUSB0"

var bmsClient *bms.DalyBMSIstance

func main() {
	fmt.Println("Starting...")

	for {
		if bmsClient != nil {
			err := bmsClient.Disconnect()
			if err != nil {
				fmt.Println("Error disconnecting from BMS: ", err)
			}
		}
		bmsClient = bms.DalyBMS()
		err := bmsClient.Connect(BMS_PORT)
		if err != nil {
			fmt.Printf("Error connecting to BMS: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		defer bmsClient.Disconnect()

		for {
			data, err := bmsClient.GetAllData()
			if err != nil {
				fmt.Println("Error getting data: ", err)
				break
			} else {
				fmt.Println("SOC: ", data)
			}

			// delay before next sample
			time.Sleep(SAMPLE_INTERVAL * time.Second)
		}

		// delay before reconnecting
		time.Sleep(1 * time.Second)
	}
}
