package dalybms

import (
	"fmt"
	"time"

	"github.com/tarm/serial"
)

// BMS serial connection
type DalyBMSIstance struct {
	serialPort     *serial.Port
	requestRetries int
	latestStatus   *StatusData // cached from GetStatus()
	address        int
}

func DalyBMS() *DalyBMSIstance {
	return &DalyBMSIstance{
		requestRetries: 3, // default
		address:        4, // default for RS485
	}
}

// Connect opens the serial port. Eg "/dev/ttyUSB0"
func (bms *DalyBMSIstance) Connect(serialDevicePath string) error {
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
func (bms *DalyBMSIstance) Disconnect() error {
	if bms.serialPort != nil {
		err := bms.serialPort.Close()
		bms.serialPort = nil
		return err
	}
	return nil
}
