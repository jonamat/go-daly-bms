package dalybms

import (
	"fmt"
	"time"

	"github.com/tarm/serial"
)

// BMS status query
type StatusData struct {
	NumberOfCells              int
	NumberOfTemperatureSensors int
	IsChargerRunning           bool
	IsLoadRunning              bool
	States                     map[string]bool
	CycleCount                 int16
}

// BMS serial connection
type DalyBMS struct {
	serialPort     *serial.Port
	requestRetries int
	address        int
	latestStatus   *StatusData // cached from GetStatus()
}

// Address = 4 for USB (RS485), or 8 for Bluetooth, per Daly docs.
func NewDalyBMS(retryCount int, address int) *DalyBMS {
	if retryCount < 1 {
		retryCount = 3
	}
	// Default to 4 if not specified.
	if address == 0 {
		address = 4
	}

	return &DalyBMS{
		requestRetries: retryCount,
		address:        address,
	}
}

// Connect opens the serial port. Eg "/dev/ttyUSB0"
func (bms *DalyBMS) Connect(serialDevicePath string) error {
	portConfig := &serial.Config{
		Name:        serialDevicePath,
		Baud:        9600,
		ReadTimeout: 100 * time.Millisecond, // e.g. 100ms
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
	}

	openedPort, err := serial.OpenPort(portConfig)
	if err != nil {
		return fmt.Errorf("failed to open serial port: %w", err)
	}

	bms.serialPort = openedPort

	// Optionally fetch initial status once connected
	_, _ = bms.GetStatus()
	return nil
}

// Close serial port
func (bms *DalyBMS) Disconnect() error {
	if bms.serialPort != nil {
		err := bms.serialPort.Close()
		bms.serialPort = nil
		return err
	}
	return nil
}
