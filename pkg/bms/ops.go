package dalybms

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
)

// Get BMS status
func (bms *DalyBMS) GetStatus() (*StatusData, error) {
	response, err := bms.sendReadRequest("94", "", 1, false)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("no data received for get_status")
	}

	responseBytes, ok := response.([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected response type for get_status")
	}
	if len(responseBytes) < 8 {
		return nil, fmt.Errorf("insufficient data length for get_status")
	}

	// Equivalent to Python struct.unpack('>b b ? ? b h x')
	var raw struct {
		Cells              int8
		TemperatureSensors int8
		ChargerRunning     bool
		LoadRunning        bool
		StateBits          int8
		CycleCount         int16
		Skip               byte
	}
	if err := binary.Read(bytes.NewReader(responseBytes), binary.BigEndian, &raw); err != nil {
		return nil, err
	}

	// Interpret the individual bits in raw.StateBits
	stateNames := []string{"DI1", "DI2", "DI3", "DI4", "DO1", "DO2", "DO3", "DO4"}
	statesMap := make(map[string]bool)
	for bitIndex := 0; bitIndex < 8; bitIndex++ {
		bitValue := (raw.StateBits >> bitIndex) & 1
		statesMap[stateNames[bitIndex]] = (bitValue == 1)
	}

	bms.latestStatus = &StatusData{
		NumberOfCells:              int(raw.Cells),
		NumberOfTemperatureSensors: int(raw.TemperatureSensors),
		IsChargerRunning:           raw.ChargerRunning,
		IsLoadRunning:              raw.LoadRunning,
		States:                     statesMap,
		CycleCount:                 raw.CycleCount,
	}
	return bms.latestStatus, nil
}

// Get State of Charge
func (bms *DalyBMS) GetSOC() (map[string]float64, error) {
	response, err := bms.sendReadRequest("90", "", 1, false)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("no data received for get_soc")
	}

	responseBytes, ok := response.([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected response type for get_soc")
	}
	if len(responseBytes) < 8 {
		return nil, fmt.Errorf("insufficient data length for get_soc")
	}

	// struct.unpack('>h h h h') => 4 big-endian int16
	var raw [4]int16
	if err := binary.Read(bytes.NewReader(responseBytes), binary.BigEndian, &raw); err != nil {
		return nil, err
	}

	return map[string]float64{
		"total_voltage": float64(raw[0]) / 10.0,
		"current":       float64(raw[2]-30000) / 10.0, // negative => charging
		"soc_percent":   float64(raw[3]) / 10.0,
	}, nil
}

// Get highest/lowest cell voltages
func (bms *DalyBMS) GetCellVoltageRange() (map[string]interface{}, error) {
	response, err := bms.sendReadRequest("91", "", 1, false)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("no data for get_cell_voltage_range")
	}

	responseBytes, ok := response.([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected type for get_cell_voltage_range")
	}
	if len(responseBytes) < 8 {
		return nil, fmt.Errorf("insufficient length for cell voltage range data")
	}

	// struct.unpack('>h b h b 2x')
	var raw struct {
		HighestVoltageRaw int16
		HighestCellID     int8
		LowestVoltageRaw  int16
		LowestCellID      int8
		Skipped           [2]byte
	}
	if err := binary.Read(bytes.NewReader(responseBytes), binary.BigEndian, &raw); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"highest_voltage": float64(raw.HighestVoltageRaw) / 1000.0,
		"highest_cell":    raw.HighestCellID,
		"lowest_voltage":  float64(raw.LowestVoltageRaw) / 1000.0,
		"lowest_cell":     raw.LowestCellID,
	}, nil
}

// Get overall highest/lowest temperature info
func (bms *DalyBMS) GetTemperatureRange() (map[string]interface{}, error) {
	response, err := bms.sendReadRequest("92", "", 1, false)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("no data for get_temperature_range")
	}

	responseBytes, ok := response.([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected type for get_temperature_range")
	}
	if len(responseBytes) < 8 {
		return nil, fmt.Errorf("insufficient length for temperature range data")
	}

	// struct.unpack('>b b b b 4x')
	var raw struct {
		HighestTemperatureRaw int8
		HighestSensor         int8
		LowestTemperatureRaw  int8
		LowestSensor          int8
		Skipped               [4]byte
	}
	if err := binary.Read(bytes.NewReader(responseBytes), binary.BigEndian, &raw); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"highest_temperature": float64(raw.HighestTemperatureRaw) - 40.0,
		"highest_sensor":      raw.HighestSensor,
		"lowest_temperature":  float64(raw.LowestTemperatureRaw) - 40.0,
		"lowest_sensor":       raw.LowestSensor,
	}, nil
}

// Get MOSFET charging/discharging status
func (bms *DalyBMS) GetMosfetStatus() (map[string]interface{}, error) {
	response, err := bms.sendReadRequest("93", "", 1, false)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("no data for get_mosfet_status")
	}

	responseBytes, ok := response.([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected type for get_mosfet_status")
	}
	if len(responseBytes) < 8 {
		return nil, fmt.Errorf("insufficient length for mosfet status")
	}

	// struct.unpack('>b ? ? B l') => int8, bool, bool, uint8, int32
	var raw struct {
		ModeRaw           int8
		ChargingMosfet    bool
		DischargingMosfet bool
		UnusedByte        uint8
		CapacityRaw       int32
	}
	if err := binary.Read(bytes.NewReader(responseBytes), binary.BigEndian, &raw); err != nil {
		return nil, err
	}

	modeText := "discharging"
	if raw.ModeRaw == 0 {
		modeText = "stationary"
	} else if raw.ModeRaw == 1 {
		modeText = "charging"
	}

	return map[string]interface{}{
		"mode":               modeText,
		"charging_mosfet":    raw.ChargingMosfet,
		"discharging_mosfet": raw.DischargingMosfet,
		"capacity_ah":        float64(raw.CapacityRaw) / 1000.0,
	}, nil
}

// Get individual cell voltages
func (bms *DalyBMS) GetCellVoltages() (map[int]float64, error) {
	maxResp, err := bms.calculateNumberOfResponses("cells", 3)
	if err != nil {
		return nil, err
	}

	response, err := bms.sendReadRequest("95", "", maxResp, true)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("no data for get_cell_voltages")
	}

	dataFrames, ok := response.([][]byte)
	if !ok {
		// maybe there is only one frame as []byte
		singleFrame, singleOk := response.([]byte)
		if singleOk {
			dataFrames = [][]byte{singleFrame}
		} else {
			return nil, fmt.Errorf("unexpected response type for get_cell_voltages")
		}
	}

	parsedValues, err := bms.splitFramesForData(dataFrames, "cells", 3)
	if err != nil {
		return nil, err
	}

	// convert each raw millivolt reading to volts
	for index, millivolts := range parsedValues {
		parsedValues[index] = millivolts / 1000.0
	}
	return parsedValues, nil
}

// Get temperature sensor values
func (bms *DalyBMS) GetTemperatures() (map[int]float64, error) {
	maxResp, err := bms.calculateNumberOfResponses("temperature_sensors", 7)
	if err != nil {
		return nil, err
	}

	response, err := bms.sendReadRequest("96", "", maxResp, true)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("no data for get_temperatures")
	}

	dataFrames, ok := response.([][]byte)
	if !ok {
		singleFrame, singleOk := response.([]byte)
		if singleOk {
			dataFrames = [][]byte{singleFrame}
		} else {
			return nil, fmt.Errorf("unexpected response type for get_temperatures")
		}
	}

	parsedValues, err := bms.splitFramesForData(dataFrames, "temperature_sensors", 7)
	if err != nil {
		return nil, err
	}

	// temperatures are raw_value - 40
	for index, rawValue := range parsedValues {
		parsedValues[index] = rawValue - 40.0
	}
	return parsedValues, nil
}

// Get cell balancing (on/off) for each cell
func (bms *DalyBMS) GetBalancingStatus() (map[int]bool, error) {
	response, err := bms.sendReadRequest("97", "", 1, false)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("no data for get_balancing_status")
	}

	responseBytes, ok := response.([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected response type for get_balancing_status")
	}

	numberOfCells := 0
	if bms.latestStatus != nil {
		numberOfCells = bms.latestStatus.NumberOfCells
	}
	balancingMap := make(map[int]bool)

	// convert entire response to a single big-endian integer, then interpret bits from the right side.
	bigIntValue := bigEndianToUint64(responseBytes)
	binaryString := fmt.Sprintf("%b", bigIntValue)
	// pad to at least 48 bits (like the Python code did zfill(48))
	for len(binaryString) < 48 {
		binaryString = "0" + binaryString
	}

	// for each cell from 1..n, check the bit from the right.
	// python code uses bits[-cellIndex].
	for cellIndex := 1; cellIndex <= numberOfCells; cellIndex++ {
		bitPosition := len(binaryString) - cellIndex
		if bitPosition < 0 {
			// no more bits to read
			break
		}
		balancingMap[cellIndex] = (binaryString[bitPosition] == '1')
	}

	return balancingMap, nil
}

// Get errors from the BMS
func (bms *DalyBMS) GetErrors() ([]string, error) {
	response, err := bms.sendReadRequest("98", "", 1, false)
	if err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("no data for get_errors")
	}

	responseBytes, ok := response.([]byte)
	if !ok {
		return nil, fmt.Errorf("unexpected response type for get_errors")
	}

	// if all zero => no errors
	isAllZero := true
	for _, singleByte := range responseBytes {
		if singleByte != 0 {
			isAllZero = false
			break
		}
	}
	if isAllZero {
		return []string{}, nil
	}

	var foundErrors []string
	for byteIndex, singleByte := range responseBytes {
		if singleByte == 0 {
			continue
		}
		// Check each bit in this byte
		for bitPos := 0; bitPos < 8; bitPos++ {
			bitMask := byte(1 << bitPos)
			if (singleByte & bitMask) != 0 {
				// The Python code looks up dalyErrorCodes[byteIndex][bitPos]
				if errorList, ok := DalyErrorCodes[byteIndex]; ok {
					if bitPos < len(errorList) {
						foundErrors = append(foundErrors, errorList[bitPos])
					} else {
						foundErrors = append(foundErrors,
							fmt.Sprintf("Unknown error code at byte=%d bit=%d", byteIndex, bitPos))
					}
				} else {
					foundErrors = append(foundErrors,
						fmt.Sprintf("Unknown error code at byte=%d bit=%d", byteIndex, bitPos))
				}
			}
		}
	}
	return foundErrors, nil
}

// Get all data in one call
func (bms *DalyBMS) FetchAllData() (map[string]interface{}, error) {
	socData, socErr := bms.GetSOC()
	if socErr != nil {
		return nil, socErr
	}

	voltageRangeData, voltageRangeErr := bms.GetCellVoltageRange()
	if voltageRangeErr != nil {
		return nil, voltageRangeErr
	}

	temperatureRangeData, temperatureRangeErr := bms.GetTemperatureRange()
	if temperatureRangeErr != nil {
		return nil, temperatureRangeErr
	}

	mosfetStatusData, mosfetStatusErr := bms.GetMosfetStatus()
	if mosfetStatusErr != nil {
		return nil, mosfetStatusErr
	}

	statusData, statusErr := bms.GetStatus()
	if statusErr != nil {
		return nil, statusErr
	}

	individualCellVoltages, cellVoltErr := bms.GetCellVoltages()
	if cellVoltErr != nil {
		return nil, cellVoltErr
	}

	temperatureSensors, tempErr := bms.GetTemperatures()
	if tempErr != nil {
		return nil, tempErr
	}

	balancingInfo, balErr := bms.GetBalancingStatus()
	if balErr != nil {
		return nil, balErr
	}

	errorsList, errorsErr := bms.GetErrors()
	if errorsErr != nil {
		return nil, errorsErr
	}

	return map[string]interface{}{
		"soc":                socData,
		"cell_voltage_range": voltageRangeData,
		"temperature_range":  temperatureRangeData,
		"mosfet_status":      mosfetStatusData,
		"status":             statusData,
		"cell_voltages":      individualCellVoltages,
		"temperatures":       temperatureSensors,
		"balancing_status":   balancingInfo,
		"errors":             errorsList,
	}, nil
}

func (bms *DalyBMS) EnableChargeMosfet(isOn bool) error {
	extraBytesHex := "00"
	if isOn {
		extraBytesHex = "01"
	}

	response, err := bms.sendReadRequest("da", extraBytesHex, 1, false)
	if err != nil {
		return err
	}
	if response == nil {
		return fmt.Errorf("no response from EnableChargeMosfet")
	}
	log.Printf("EnableChargeMosfet response: %x\n", response)
	return nil
}

func (bms *DalyBMS) EnableDischargeMosfet(isOn bool) error {
	extraBytesHex := "00"
	if isOn {
		extraBytesHex = "01"
	}

	response, err := bms.sendReadRequest("d9", extraBytesHex, 1, false)
	if err != nil {
		return err
	}
	if response == nil {
		return fmt.Errorf("no response from EnableDischargeMosfet")
	}
	log.Printf("EnableDischargeMosfet response: %x\n", response)
	return nil
}

// Set SoC percentage (0..100)
func (bms *DalyBMS) SetSOC(socPercent float64) error {
	rawValue := int(socPercent * 10.0)
	if rawValue > 1000 {
		rawValue = 1000
	}
	if rawValue < 0 {
		rawValue = 0
	}

	// Format: '000000000000%04X'
	extraBytesHex := fmt.Sprintf("000000000000%04X", rawValue)

	response, err := bms.sendReadRequest("21", extraBytesHex, 1, false)
	if err != nil {
		return err
	}
	if response == nil {
		return fmt.Errorf("no response from SetSOC")
	}
	log.Printf("SetSOC response: %x\n", response)
	return nil
}

// Restart device. The effect may depend on device firmware.
func (bms *DalyBMS) Restart() error {
	response, err := bms.readSerialResponse("00", "", 1, false)
	if err != nil {
		return err
	}
	if response == nil {
		return fmt.Errorf("no response from Restart")
	}
	log.Printf("Restart response: %v\n", response)
	return nil
}
