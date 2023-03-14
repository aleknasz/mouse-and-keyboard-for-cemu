package controller

import (
	"encoding/binary"
	"encoding/hex"
	"hash/crc32"
	"math"
	"strings"
)

type Vector3 struct {
	Z float32 `json:"z"`
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

var ZeroVector3 = Vector3{0.0, 0.0, 0.0}

const (
	DSUC_VersionReq = 0x100000
	DSUS_VersionRsp = 0x100000
	DSUC_ListPorts  = 0x100001
	DSUS_PortInfo   = 0x100001
	DSUC_PadDataReq = 0x100002
	DSUS_PadDataRsp = 0x100002
)

const MaxProtocolVer = 1001
const ServerID = 1423567

var packetCounter uint32 = 0

type DSUProtocol struct {
	MessageType      uint32
	NumOfPadRequests int
	Flags            byte
	IdToRRegister    byte
	MacToRegister    string
	RequestIndex     []byte
}

func (p *DSUProtocol) ReadRequest(data []byte) {
	index := 4                                               // for now skip header
	index += 2                                               // skip protocol
	index += 2                                               // skip packetSize
	index += 4                                               // skip crc
	index += 4                                               // skip client id
	p.MessageType = binary.LittleEndian.Uint32(data[index:]) // care only about message type
	index += 4
	if p.MessageType == DSUC_ListPorts {
		p.NumOfPadRequests = int(binary.LittleEndian.Uint32(data[index:]))
		p.RequestIndex = make([]byte, p.NumOfPadRequests)
		index += 4
		for i := 0; i < p.NumOfPadRequests; i += 1 {
			p.RequestIndex[i] = data[index+i]
		}
	} else if p.MessageType == DSUC_PadDataReq {
		p.Flags = data[index]
		index += 1
		p.IdToRRegister = data[index]
		index += 1
		macToRegisterBuf := make([]string, 6)
		for i := 0; i < 6; i += 1 {
			if data[index] >= 15 {
				macToRegisterBuf[i] = hex.EncodeToString(data[index : index+1])
			} else {
				macToRegisterBuf[i] = "00"
			}
			index += 1
		}
		p.MacToRegister = strings.Join(macToRegisterBuf, ":")
	}
}

func (p *DSUProtocol) CreateListPortsResponse() []byte {

	outBuffer := make([]byte, 32) // plus 16 for header
	createPacketHeader(outBuffer)
	binary.LittleEndian.PutUint32(outBuffer[16:], DSUS_PortInfo)
	outIndex := 4
	outBuffer[outIndex] = 0x00 // slot
	outIndex += 1
	outBuffer[outIndex] = 0x02 // state (connected)
	outIndex += 1
	outBuffer[outIndex] = 0x02 // model (generic)
	outIndex += 1
	outBuffer[outIndex] = 0x01 // connection type (usb)
	outIndex += 1

	// Set MAC address 00:00:00:00:00:FF
	for j := 0; j < 5; j += 1 {
		outBuffer[outIndex] = 0x00
		outIndex += 1
	}
	outBuffer[outIndex] = 0xFF
	outIndex += 1

	outBuffer[outIndex] = 0xEF // battery (charged)
	outIndex += 1
	outBuffer[outIndex] = 0 // n/a
	outIndex += 1

	calculateChecksum(outBuffer)
	return outBuffer

}

func (p *DSUProtocol) CreateControllerResponse(userController *ControllerState, motionTimestamp uint64, gyro Vector3) []byte {

	outBuffer := make([]byte, 100)
	createPacketHeader(outBuffer)
	outIndex := 16
	binary.LittleEndian.PutUint32(outBuffer[outIndex:], DSUS_PadDataRsp)
	outIndex += 4

	outBuffer[outIndex] = 0x00 // pad id
	outIndex += 1
	outBuffer[outIndex] = 0x02 // state (connected)
	outIndex += 1
	outBuffer[outIndex] = 0x02 // model (generic)
	outIndex += 1
	outBuffer[outIndex] = 0x01 // connection type (usb)
	outIndex += 1

	for j := 0; j < 5; j += 1 {
		outBuffer[outIndex] = 0x00
		outIndex += 1
	}
	outBuffer[outIndex] = 0xFF
	outIndex += 1

	outBuffer[outIndex] = 0xEF // battery (charged)
	outIndex += 1
	outBuffer[outIndex] = 0x01 // is active (true)
	outIndex += 1

	binary.LittleEndian.PutUint32(outBuffer[outIndex:], packetCounter)
	packetCounter += 1
	outIndex += 4

	outBuffer[outIndex] = userController.GetDPadMask() // D-Pad Left, D-Pad Down, D-Pad Right, D-Pad Up, Options (?), R3, L3, Share (?)
	outIndex += 1
	outBuffer[outIndex] = userController.GetButtonMask() // Y, B, A, X, R1, L1, R2, L2
	outIndex += 1
	outBuffer[outIndex] = 0x00 // HOME Button (0 or 1)
	outIndex += 1
	outBuffer[outIndex] = 0x00 // Touch Button (0 or 1)
	outIndex += 1

	outBuffer[outIndex] = userController.GetStickMask(L_STICK, X_AXIS) // Left stick X (plus rightward)
	outIndex += 1
	outBuffer[outIndex] = userController.GetStickMask(L_STICK, Y_AXIS) // Left stick Y (plus upward)
	outIndex += 1
	x_axis := userController.GetStickValue(R_STICK, X_AXIS)

	outBuffer[outIndex] = x_axis // Right stick X (plus rightward)
	outIndex += 1
	y_axis := userController.GetStickValue(R_STICK, Y_AXIS)
	outBuffer[outIndex] = y_axis // Right stick Y (plus upward)
	outIndex += 1

	//log.Printf("Mouse move %d %d", x_axis, y_axis)

	outBuffer[outIndex] = 0x00 // Analog D-Pad Left
	outIndex += 1
	outBuffer[outIndex] = 0x00 // Analog D-Pad Down
	outIndex += 1
	outBuffer[outIndex] = 0x00 // Analog D-Pad Right
	outIndex += 1
	outBuffer[outIndex] = 0x00 // Analog D-Pad Up
	outIndex += 1

	outBuffer[outIndex] = 0x00 // Analog Y
	outIndex += 1
	outBuffer[outIndex] = 0x00 // Analog B
	outIndex += 1
	outBuffer[outIndex] = 0x00 // Analog A
	outIndex += 1
	outBuffer[outIndex] = 0x00 // Analog X
	outIndex += 1

	outBuffer[outIndex] = 0x00 // Analog R1
	outIndex += 1
	outBuffer[outIndex] = 0x00 // Analog L1
	outIndex += 1

	outBuffer[outIndex] = 0x00 // Analog R2
	outIndex += 1
	outBuffer[outIndex] = 0x00 // Analog L2
	outIndex += 1

	outBuffer[outIndex] = 0x00 // First touch (inactive)
	outIndex += 1
	outBuffer[outIndex] = 0x00 // First touch id
	outIndex += 1
	binary.LittleEndian.PutUint16(outBuffer[outIndex:], 0x0000) // trackpad first x
	outIndex += 2
	binary.LittleEndian.PutUint16(outBuffer[outIndex:], 0x0000) // trackpad first y
	outIndex += 2

	outBuffer[outIndex] = 0x00 // trackpad second is active (false)
	outIndex += 1
	outBuffer[outIndex] = 0x00 // trackpad second id
	outIndex += 1
	binary.LittleEndian.PutUint16(outBuffer[outIndex:], 0x0000) // trackpad second x
	outIndex += 2
	binary.LittleEndian.PutUint16(outBuffer[outIndex:], 0x0000) // trackpad second y
	outIndex += 2
	//

	binary.LittleEndian.PutUint64(outBuffer[outIndex:], motionTimestamp)
	//binary.LittleEndian.PutUint64(outBuffer[outIndex:], report.Ts)
	outIndex += 8

	binary.LittleEndian.PutUint32(outBuffer[outIndex:], math.Float32bits(ZeroVector3.X)) // x
	outIndex += 4
	binary.LittleEndian.PutUint32(outBuffer[outIndex:], math.Float32bits(ZeroVector3.Y)) // y
	outIndex += 4
	binary.LittleEndian.PutUint32(outBuffer[outIndex:], math.Float32bits(ZeroVector3.Z)) // z
	outIndex += 4

	binary.LittleEndian.PutUint32(outBuffer[outIndex:], math.Float32bits(gyro.X)) // x
	outIndex += 4
	binary.LittleEndian.PutUint32(outBuffer[outIndex:], math.Float32bits(gyro.Y)) // y
	outIndex += 4
	binary.LittleEndian.PutUint32(outBuffer[outIndex:], math.Float32bits(gyro.Z)) // z
	outIndex += 4

	calculateChecksum(outBuffer)

	return outBuffer
}

func createPacketHeader(data []byte) {

	copy(data, "DSUS")

	index := 4

	binary.LittleEndian.PutUint16(data[index:], uint16(MaxProtocolVer))
	index += 2

	binary.LittleEndian.PutUint16(data[index:], uint16(len(data)-16))
	index += 2

	binary.LittleEndian.PutUint32(data[index:], uint32(0))
	index += 4

	binary.LittleEndian.PutUint32(data[index:], uint32(ServerID))
	index += 4

}

func calculateChecksum(data []byte) {
	table := crc32.MakeTable(crc32.IEEE)
	checksum := crc32.Checksum(data, table)
	binary.LittleEndian.PutUint32(data[8:], checksum)
}
