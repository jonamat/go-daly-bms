package dalybms

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"time"
)

// calculateNumberOfResponses determines how many 13-byte response frames we expect
// for given data (like cells or temperature sensors).
func (bms *DalyBMS) calculateNumberOfResponses(statusField string, itemCountPerFrame int) (int, error) {
	if bms.latestStatus == nil {
		return 0, fmt.Errorf("getStatus must be called before retrieving %s", statusField)
	}

	switch statusField {
	case "cells":
		if bms.address == 8 {
			// Bluetooth returns all frames up to 16
			return 16, nil
		}
		return int(math.Ceil(float64(bms.latestStatus.NumberOfCells) / float64(itemCountPerFrame))), nil

	case "temperature_sensors":
		if bms.address == 8 {
			// Bluetooth returns up to 3 frames
			return 3, nil
		}
		return int(math.Ceil(float64(bms.latestStatus.NumberOfTemperatureSensors) / float64(itemCountPerFrame))), nil
	}

	return 0, fmt.Errorf("unknown status field: %s", statusField)
}

// splitFramesForData is a helper that unpacks multi-frame responses for cell or temperature data.
func (bms *DalyBMS) splitFramesForData(
	frames [][]byte,
	statusField string,
	itemsPerFrame int,
) (map[int]float64, error) {

	if bms.latestStatus == nil {
		return nil, fmt.Errorf("getStatus must be called before retrieving %s", statusField)
	}

	var needed int
	if statusField == "cells" {
		needed = bms.latestStatus.NumberOfCells
	} else if statusField == "temperature_sensors" {
		needed = bms.latestStatus.NumberOfTemperatureSensors
	} else {
		return nil, fmt.Errorf("unknown field: %s", statusField)
	}

	results := make(map[int]float64)
	expectedFrameIndex := 1

	for _, frame := range frames {
		if len(frame) < 1 {
			// skip
			continue
		}

		frameNumber := int(frame[0])
		if frameNumber != expectedFrameIndex {
			log.Printf("splitFramesForData warning: expected frame=%d, got frame=%d", expectedFrameIndex, frameNumber)
		}

		frameReader := bytes.NewReader(frame[1:]) // skip the frame index byte
		for itemIndex := 0; itemIndex < itemsPerFrame; itemIndex++ {
			// "cells": we read int16 each
			// "temperature_sensors": we read int8 each
			if statusField == "cells" {
				var cellValue int16
				if err := binary.Read(frameReader, binary.BigEndian, &cellValue); err != nil {
					break
				}
				results[len(results)+1] = float64(cellValue)
			} else {
				var temperatureValue int8
				if err := binary.Read(frameReader, binary.BigEndian, &temperatureValue); err != nil {
					break
				}
				results[len(results)+1] = float64(temperatureValue)
			}

			if len(results) == needed {
				// We have all items
				return results, nil
			}
		}
		expectedFrameIndex++
	}

	return results, nil
}

// sendReadRequest is a higher-level function that retries the readSerialResponse
// up to bms.requestRetries times.
func (bms *DalyBMS) sendReadRequest(
	command string,
	extraHexData string,
	maxResponses int,
	returnList bool,
) (interface{}, error) {

	var finalResult interface{}
	var finalErr error

	for attemptIndex := 0; attemptIndex < bms.requestRetries; attemptIndex++ {
		readResult, readErr := bms.readSerialResponse(command, extraHexData, maxResponses, returnList)
		if readErr != nil {
			log.Printf("Attempt %d for command %s failed: %v", attemptIndex+1, command, readErr)
			time.Sleep(200 * time.Millisecond)
			finalErr = readErr
			continue
		}
		if readResult == nil {
			log.Printf("Attempt %d for command %s returned nil response; retrying", attemptIndex+1, command)
			time.Sleep(200 * time.Millisecond)
			finalErr = fmt.Errorf("nil response")
			continue
		}
		// success
		return readResult, nil
	}
	return finalResult, fmt.Errorf("command %s failed after %d tries: %w", command, bms.requestRetries, finalErr)
}

// readSerialResponse writes a command to the BMS and attempts to read a specified
// number of 13-byte responses. If returnList is false, and we only get one response,
// we return the raw 8 data bytes. If multiple frames are returned or returnList=true,
// we return a slice of slices.
func (bms *DalyBMS) readSerialResponse(
	command string,
	extraHexData string,
	maxResponses int,
	returnList bool,
) (interface{}, error) {

	if bms.serialPort == nil {
		return nil, fmt.Errorf("serial port not open")
	}

	requestFrame, err := bms.buildRequestFrame(command, extraHexData)
	if err != nil {
		return nil, fmt.Errorf("failed to build request frame: %w", err)
	}

	// Drain any leftover data.
	if err := bms.drainReadBuffer(); err != nil {
		// not fatal, just log
		log.Printf("Warning: draining buffer: %v", err)
	}

	// Write out the command.
	bytesWritten, err := bms.serialPort.Write(requestFrame)
	if err != nil || bytesWritten != len(requestFrame) {
		return nil, fmt.Errorf("failed to write command %s to serial port", command)
	}

	var collectedData [][]byte

	// Each full response is 13 bytes: 4 for header, 8 for data, 1 for CRC
	for frameIndex := 0; frameIndex < maxResponses; frameIndex++ {
		readBuffer := make([]byte, 13)
		bytesRead, readErr := bms.serialPort.Read(readBuffer)
		if readErr != nil || bytesRead == 0 {
			// Probably a timeout or no more data
			break
		}

		if bytesRead < 13 {
			// partial read
			log.Printf("Partial response for command %s: got %d bytes (expected 13)", command, bytesRead)
			break
		}

		// Check CRC
		computedCRC := computeCRC(readBuffer[:12])
		if computedCRC != readBuffer[12] {
			log.Printf("CRC mismatch for command %s: computed %02x != %02x", command, computedCRC, readBuffer[12])
			continue
		}

		// Validate the command nibble in header
		headerHex := fmt.Sprintf("%02x%02x%02x%02x", readBuffer[0], readBuffer[1], readBuffer[2], readBuffer[3])
		if len(headerHex) >= 6 && headerHex[4:6] != command {
			log.Printf("Invalid header for command %s: got %s (mismatched command code)", command, headerHex)
			continue
		}

		// The 8 data bytes are readBuffer[4:12]
		dataBytes := readBuffer[4:12]
		collectedData = append(collectedData, dataBytes)

		if len(collectedData) == maxResponses {
			break
		}
	}

	if len(collectedData) == 0 {
		return nil, nil
	}

	// If multiple frames or returnList is explicitly requested
	if returnList || len(collectedData) > 1 {
		return collectedData, nil
	}

	// Otherwise return just the single 8-byte data section
	return collectedData[0], nil
}

// buildRequestFrame constructs the hex message plus CRC for a command like "90" with optional extra data.
// The result is a 13-byte packet: 12 bytes (in hex form) + 1-byte CRC.
func (bms *DalyBMS) buildRequestFrame(command string, extraHex string) ([]byte, error) {
	// Example: "a5[address]0[cmd]08[extra]" => pad to 24 hex digits => then 1-byte CRC.
	// e.g. "a5409008000000000000000000" + CRC => 13 total bytes.

	hexString := fmt.Sprintf("a5%x0%s08%s", bms.address, command, extraHex)

	// Pad out to 24 hex characters
	for len(hexString) < 24 {
		hexString += "0"
	}

	rawBytes, err := decodeHexString(hexString)
	if err != nil {
		return nil, fmt.Errorf("hex decode error: %w", err)
	}

	finalCRC := computeCRC(rawBytes)
	rawBytes = append(rawBytes, finalCRC)
	return rawBytes, nil
}

// drainReadBuffer attempts to read any leftover data so it doesn't mix with new responses.
func (bms *DalyBMS) drainReadBuffer() error {
	if bms.serialPort == nil {
		return fmt.Errorf("drain requested but serialPort is nil")
	}

	leftoverBuffer := make([]byte, 256)

	// Repeatedly read until .Read() returns 0 or an error,
	// meaning there's no more data immediately available in the driver buffer.
	for {
		bytesRead, readErr := bms.serialPort.Read(leftoverBuffer)
		if readErr != nil || bytesRead == 0 {
			break
		}
	}
	return nil
}

// computeCRC sums all bytes and returns the low byte of the sum.
func computeCRC(message []byte) byte {
	var sum uint32
	for _, singleByte := range message {
		sum += uint32(singleByte)
	}
	return byte(sum & 0xFF)
}

// decodeHexString decodes a hex string to raw bytes.
func decodeHexString(hexText string) ([]byte, error) {
	raw := make([]byte, len(hexText)/2)
	_, err := fmt.Sscanf(hexText, "%x", &raw)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// bigEndianToUint64 interprets a byte slice as a big-endian 64-bit integer.
func bigEndianToUint64(data []byte) uint64 {
	var val uint64
	for _, b := range data {
		val = (val << 8) | uint64(b)
	}
	return val
}
