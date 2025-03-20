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
				fmt.Println("Balancing status: ", data.BalancingStatus)
				fmt.Println("Highest cell: ", data.CellVoltageRange.HighestCell)
				fmt.Println("Lowest cell: ", data.CellVoltageRange.LowestCell)
				fmt.Println("Highest voltage: ", data.CellVoltageRange.HighestVoltage)
				fmt.Println("Lowest voltage: ", data.CellVoltageRange.LowestVoltage)
				fmt.Println("Cell voltages: ", data.CellVoltages)
				fmt.Println("Errors: ", data.Errors)
				fmt.Println("Capacity Ah: ", data.MosfetStatus.CapacityAh)
				fmt.Println("Charging mosfet: ", data.MosfetStatus.ChargingMosfet)
				fmt.Println("Discharging mosfet: ", data.MosfetStatus.DischargingMosfet)
				fmt.Println("Mode: ", data.MosfetStatus.Mode)
				fmt.Println("Current: ", data.SOC.Current)
				fmt.Println("SOC percent: ", data.SOC.SOCPercent)
				fmt.Println("Total voltage: ", data.SOC.TotalVoltage)
				fmt.Println("Cycle count: ", data.Status.CycleCount)
				fmt.Println("Is charger running: ", data.Status.IsChargerRunning)
				fmt.Println("Is load running: ", data.Status.IsLoadRunning)
				fmt.Println("Number of cells: ", data.Status.NumberOfCells)
				fmt.Println("Number of temperature sensors: ", data.Status.NumberOfTemperatureSensors)
				fmt.Println("States: ", data.Status.States)
				fmt.Println("Highest sensor: ", data.TemperatureRange.HighestSensor)
				fmt.Println("Lowest sensor: ", data.TemperatureRange.LowestSensor)
				fmt.Println("Highest temperature: ", data.TemperatureRange.HighestTemperature)
				fmt.Println("Lowest temperature: ", data.TemperatureRange.LowestTemperature)
			}

			// delay before next sample
			time.Sleep(SAMPLE_INTERVAL * time.Second)
		}

		// delay before reconnecting
		time.Sleep(1 * time.Second)
	}
}

/*
	Output example:
	Starting...
	Balancing status:  map[1:false 2:false 3:false 4:false]
	Highest cell:  3
	Lowest cell:  1
	Highest voltage:  3.279
	Lowest voltage:  3.255
	Cell voltages:  map[1:3.255 2:3.279 3:3.279 4:3.259]
	Errors:  []
	Capacity Ah:  147.43
	Charging mosfet:  true
	Discharging mosfet:  true
	Mode:  stationary
	Current:  0
	SOC percent:  64.1
	Total voltage:  13
	Cycle count:  273
	Is charger running:  false
	Is load running:  false
	Number of cells:  4
	Number of temperature sensors:  1
	States:  map[DI1:false DI2:true DI3:false DI4:false DO1:false DO2:false DO3:false DO4:false]
	Highest sensor:  1
	Lowest sensor:  1
	Highest temperature:  13
	Lowest temperature:  13
*/
