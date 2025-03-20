package dalybms

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
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

// Get BMS status
func (bms *DalyBMSIstance) GetStatus() (*StatusData, error) {
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

type SOCData struct {
	TotalVoltage float32
	Current      float32
	SOCPercent   float32
}

// Get State of Charge
func (bms *DalyBMSIstance) GetSOC() (*SOCData, error) {
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

	socData := &SOCData{
		TotalVoltage: float32(raw[0]) / 10.0,
		Current:      float32(raw[2]-30000) / 10.0,
		SOCPercent:   float32(raw[3]) / 10.0,
	}

	return socData, nil
}

type CellVoltageRangeData struct {
	HighestVoltage float32
	HighestCell    int8
	LowestVoltage  float32
	LowestCell     int8
}

// Get highest/lowest cell voltages
func (bms *DalyBMSIstance) GetCellVoltageRange() (*CellVoltageRangeData, error) {
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

	cellVoltageRangeData := &CellVoltageRangeData{
		HighestVoltage: float32(raw.HighestVoltageRaw) / 1000.0,
		HighestCell:    raw.HighestCellID,
		LowestVoltage:  float32(raw.LowestVoltageRaw) / 1000.0,
		LowestCell:     raw.LowestCellID,
	}

	return cellVoltageRangeData, nil
}

type TemperatureRangeData struct {
	HighestTemperature float32
	HighestSensor      int8
	LowestTemperature  float32
	LowestSensor       int8
}

// Get overall highest/lowest temperature info
func (bms *DalyBMSIstance) GetTemperatureRange() (*TemperatureRangeData, error) {
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

	temperatureRangeData := &TemperatureRangeData{
		HighestTemperature: float32(raw.HighestTemperatureRaw) - 40.0,
		HighestSensor:      raw.HighestSensor,
		LowestTemperature:  float32(raw.LowestTemperatureRaw) - 40.0,
		LowestSensor:       raw.LowestSensor,
	}

	return temperatureRangeData, nil
}

type MosfetStatusData struct {
	Mode              string
	ChargingMosfet    bool
	DischargingMosfet bool
	CapacityAh        float32
}

// Get MOSFET charging/discharging status
func (bms *DalyBMSIstance) GetMosfetStatus() (*MosfetStatusData, error) {
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

	mosfetStatusData := &MosfetStatusData{
		Mode:              modeText,
		ChargingMosfet:    raw.ChargingMosfet,
		DischargingMosfet: raw.DischargingMosfet,
		CapacityAh:        float32(raw.CapacityRaw) / 1000.0,
	}

	return mosfetStatusData, nil
}

// Get individual cell voltages in a map[cellIndex] = voltage
func (bms *DalyBMSIstance) GetCellVoltages() (map[int]float64, error) {
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

// Get temperature sensor values in a map[sensorIndex] = temperature
func (bms *DalyBMSIstance) GetTemperatures() (map[int]float64, error) {
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

// Get cell balancing (on/off) for each cell in a map[cellIndex] = isBalancing
func (bms *DalyBMSIstance) GetBalancingStatus() (map[int]bool, error) {
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
func (bms *DalyBMSIstance) GetErrors() ([]string, error) {
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

type AllBMSData struct {
	SOC              *SOCData
	CellVoltageRange *CellVoltageRangeData
	TemperatureRange *TemperatureRangeData
	MosfetStatus     *MosfetStatusData
	Status           *StatusData
	CellVoltages     map[int]float64
	Temperatures     map[int]float64
	BalancingStatus  map[int]bool
	Errors           []string
}

// Get all data in one call
func (bms *DalyBMSIstance) GetAllData() (*AllBMSData, error) {
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

	allBmsData := &AllBMSData{
		SOC:              socData,
		CellVoltageRange: voltageRangeData,
		TemperatureRange: temperatureRangeData,
		MosfetStatus:     mosfetStatusData,
		Status:           statusData,
		CellVoltages:     individualCellVoltages,
		Temperatures:     temperatureSensors,
		BalancingStatus:  balancingInfo,
		Errors:           errorsList,
	}

	return allBmsData, nil
}

// Enable charge MOSFET switch (if on, the BMS will allow charging)
func (bms *DalyBMSIstance) EnableChargeMosfet(isOn bool) error {
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

// Enable discharge MOSFET switch (if on, the BMS will allow discharging)
func (bms *DalyBMSIstance) EnableDischargeMosfet(isOn bool) error {
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
func (bms *DalyBMSIstance) SetSOC(socPercent float64) error {
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
func (bms *DalyBMSIstance) Restart() error {
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
