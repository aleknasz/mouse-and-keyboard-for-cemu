package main

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	robot "github.com/go-vgo/robotgo"
	websocket "github.com/gorilla/websocket"
	hook "github.com/robotn/gohook"
	"hash/crc32"
	"log"
	"math"
	"math/rand"
	"mouse-and-keyboard-for-cemu/controller"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const maxProtocolVer = 1001
const serverID = 1423567
const clientTimeoutLimit = 30000

var lastRequestAt = time.Now().UnixMilli()
var connectedClient net.Addr = nil
var packetCounter uint32 = 0
var webSocketClient *websocket.Conn = nil

// var report PhoneReport
var udpServer net.PacketConn

var frameCount = 0
var fps = int64(60)
var fpsInterval = 1000 / fps
var then = time.Now().UnixMilli()
var startTime = then

type Vector3 struct {
	Z float32 `json:"z"`
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

type PhoneReport struct {
	Ts   string  `json:"ts"`
	Gyro Vector3 `json:"gyro"`
}

var zeroVector3 = Vector3{0.0, 0.0, 0.0}

const (
	DSUC_VersionReq = 0x100000
	DSUS_VersionRsp = 0x100000
	DSUC_ListPorts  = 0x100001
	DSUS_PortInfo   = 0x100001
	DSUC_PadDataReq = 0x100002
	DSUS_PadDataRsp = 0x100002
)

func BeginPacket(data []byte) {

	copy(data, "DSUS")

	index := 4

	binary.LittleEndian.PutUint16(data[index:], uint16(maxProtocolVer))
	index += 2

	binary.LittleEndian.PutUint16(data[index:], uint16(len(data)-16))
	index += 2

	binary.LittleEndian.PutUint32(data[index:], uint32(0))
	index += 4

	binary.LittleEndian.PutUint32(data[index:], uint32(serverID))
	index += 4

}

func FinishPacket(data []byte) {
	table := crc32.MakeTable(crc32.IEEE)
	checksum := crc32.Checksum(data, table)
	binary.LittleEndian.PutUint32(data[8:], checksum)
}

func SendPacket(udpServer net.PacketConn, addr net.Addr, buffer []byte) {
	BeginPacket(buffer)
	FinishPacket(buffer)
	udpServer.WriteTo(buffer, addr)
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}
	fmt.Printf("Web clinet connected\n")
	webSocketClient = ws
}

func wsReader(conn *websocket.Conn) {
	for {
		// read in a message
		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			continue
		}

		var now = time.Now().UnixMilli()
		var elapsed = now - then

		if elapsed <= fpsInterval {
			fmt.Printf("Too fast %d, expected %d\n", elapsed, fpsInterval)
			continue
		}

		//fmt.Printf("OK %d, expected %d\n", elapsed, fpsInterval)

		then = now - (elapsed % fpsInterval)

		// print out that message for clarity

		var report PhoneReport
		err = json.Unmarshal(data, &report)
		if err != nil {
			log.Println(err)
			continue
		}

		motionTimestamp, err := strconv.ParseUint(report.Ts, 10, 64)
		if err != nil {
			fmt.Printf("Error %v\n", err)
			return
		}

		/*[]byte(`{"ts":"1677354485344000", "gyro" : {"z":3.3425002, "x":2.66, "y": 0.67375004}}`)*/
		Report(udpServer, motionTimestamp, zeroVector3, report.Gyro)
		//
		//if err := conn.WriteMessage(messageType, p); err != nil {
		//	log.Println(err)
		//	return
		//}

	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func homePage(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func handleUDP(udpServer net.PacketConn) {
	data := make([]byte, 512)
	for {

		_, addr, err := udpServer.ReadFrom(data)
		if err != nil {
			continue
		}

		length := len(data)

		//fmt.Printf("Received %d bytes, elapsed %v, expected %v\n", length)

		if length == 0 {
			fmt.Printf("Empty receive\n")
			continue
		}

		index := 4                                              // for now skip header
		index += 2                                              // skip protocol
		index += 2                                              // skip packetSize
		index += 4                                              // skip crc
		index += 4                                              // skip client id
		messageType := binary.LittleEndian.Uint32(data[index:]) // care only about message type
		index += 4

		if messageType == DSUC_VersionReq {
			//fmt.Printf("Version request\n")
		} else if messageType == DSUC_ListPorts {
			fmt.Printf("List ports\n")
			numOfPadRequests := int(binary.LittleEndian.Uint32(data[index:]))
			index += 4
			for i := 0; i < numOfPadRequests; i += 1 {
				requestIndex := data[index+i]
				if requestIndex != 0 {
					continue
				}
				outBuffer := make([]byte, 32) // plus 16 for header
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

				go SendPacket(udpServer, addr, outBuffer)
			}
		} else if messageType == DSUC_PadDataReq {
			flags := data[index]
			index += 1
			idToRRegister := data[index]
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
			macToRegister := strings.Join(macToRegisterBuf, ":")
			//fmt.Printf("Pad data request for %s with flags %d and id %d\n", macToRegister, flags,
			//	idToRRegister)
			if (flags == 0 || (idToRRegister == 0 && (flags&0x01) != 0)) ||
				((macToRegister == "00:00:00:00:00:ff") && (flags&0x02) != 0) {
				lastRequestAt = time.Now().UnixMilli()
				if connectedClient == nil {
					fmt.Printf("Game connected from %v at %d\n", addr, lastRequestAt)
				}
				connectedClient = addr
			}
		}
	}
}

func main() {

	//var outBuffer = make([]byte, 100)
	//now := 1677329375368994 //time.Now().UnixMicro()
	// time.Now().UnixMilli()
	//binary.LittleEndian.PutUint32(outBuffer, math.Float32bits(-123.45))
	//binary.LittleEndian.PutUint64(outBuffer, uint64(now))
	//fmt.Printf("Wrote %s %d %d\n", hex.EncodeToString(outBuffer), time.Now().UnixMicro(), now)

	us, err := net.ListenPacket("udp", ":26760")
	if err != nil {
		panic(err)
	}
	udpServer = us
	defer udpServer.Close()

	go func() {
		http.HandleFunc("/", homePage)
		http.HandleFunc("/ws", wsEndpoint)
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	rand.Seed(time.Now().UnixNano())

	chanHook := hook.Start()
	defer hook.End()

	go captureEvents(udpServer, chanHook)

	handleUDP(udpServer)

}

func Report(udpServer net.PacketConn, motionTimestamp uint64, accelerometer Vector3, gyro Vector3) {
	client := connectedClient
	if client == nil {
		return
	}

	if time.Now().UnixMilli()-lastRequestAt > clientTimeoutLimit {
		fmt.Printf("Game timeout %s, disconnecting\n", client)
		connectedClient = nil
		return
	}

	//fmt.Printf("Unmarshalled %v\n", report)

	//if report.Ts == "" {
	//	return
	//}

	outBuffer := make([]byte, 100)
	BeginPacket(outBuffer)
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

	outBuffer[outIndex] = 0x00 // D-Pad Left, D-Pad Down, D-Pad Right, D-Pad Up, Options (?), R3, L3, Share (?)
	outIndex += 1
	outBuffer[outIndex] = 0x00
	outIndex += 1
	outBuffer[outIndex] = 0x00 // HOME Button (0 or 1)
	outIndex += 1
	outBuffer[outIndex] = 0x00 // Touch Button (0 or 1)
	outIndex += 1

	outBuffer[outIndex] = 0x80 // Left stick X (plus rightward)
	outIndex += 1
	outBuffer[outIndex] = 0x80 // Left stick Y (plus upward)
	outIndex += 1
	outBuffer[outIndex] = 0x80 // Right stick X (plus rightward)
	outIndex += 1
	outBuffer[outIndex] = 0x80 // Right stick Y (plus upward)
	outIndex += 1

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

	binary.LittleEndian.PutUint32(outBuffer[outIndex:], math.Float32bits(zeroVector3.X)) // x
	outIndex += 4
	binary.LittleEndian.PutUint32(outBuffer[outIndex:], math.Float32bits(zeroVector3.Y)) // y
	outIndex += 4
	binary.LittleEndian.PutUint32(outBuffer[outIndex:], math.Float32bits(zeroVector3.Z)) // z
	outIndex += 4

	binary.LittleEndian.PutUint32(outBuffer[outIndex:], math.Float32bits(gyro.X)) // x
	outIndex += 4
	binary.LittleEndian.PutUint32(outBuffer[outIndex:], math.Float32bits(gyro.Y)) // y
	outIndex += 4
	binary.LittleEndian.PutUint32(outBuffer[outIndex:], math.Float32bits(gyro.Z)) // z
	outIndex += 4

	FinishPacket(outBuffer)

	//fmt.Printf("Send: %s at %d\n",
	//	hex.EncodeToString(outBuffer), motionTimestamp)

	//fmt.Printf("Send package to %s at %d\n", client, lastRequestAt)

	udpServer.WriteTo(outBuffer, client)
}

func captureEvents(udpServer net.PacketConn, chanHook <-chan hook.Event) {
	var prevX, prevY int16 = -1, -1
	var sensitivity float32 = 25.0
	var mouseSwitch bool = false
	for ev := range chanHook {

		if ev.Kind == hook.MouseMove {

			if prevX == -1 {
				prevX = ev.X
			}

			if prevY == -1 {
				prevY = ev.Y
			}

			var yaw = float32(prevX - ev.X)
			var pitch = float32(prevY - ev.Y)

			prevX = ev.X
			prevY = ev.Y
			// 1440 x 900

			//if yaw == 0.0 && pitch == 0.0 {
			//	fmt.Printf("Skip %d\n", packetCounter)
			//	Report(udpServer, 0, 0, 0)
			//} else {
			//	fmt.Printf("Pos: %d %d %d\n", yaw, pitch, packetCounter)
			//	Report(udpServer, yaw, pitch, time.Now().UnixMilli())
			//}

			var gyro = Vector3{0.0, sensitivity * pitch, sensitivity * -yaw}

			if mouseSwitch {
				sx, sy := robot.GetScreenSize()
				robot.Move(sx/2, sy/2)
				//fmt.Printf("Mouse event: %v\n", gyro)
			}

			if webSocketClient != nil {
				webSocketClient.WriteJSON(Vector3{0.0,
					float32(ev.X), float32(ev.Y)})
			}

			Report(udpServer, uint64(time.Now().UnixMicro()), zeroVector3, gyro)

		} else if ev.Kind == hook.KeyUp {

			key := controller.Raw2Keycode[ev.Rawcode]

			fmt.Printf("\nKey Up: %s %d\n", key.Name, ev.Rawcode)

			//	fmt.Printf("key up: rawcode=%d rawcode=0x%x keycode=%d keycode=0x%x keychar=%d keychar=0x%x\n\n",
			//		ev.Rawcode, ev.Rawcode, ev.Keycode, ev.Keycode, ev.Keychar, ev.Keychar)
		} else if ev.Kind == hook.KeyDown {
			key := controller.Raw2Keycode[ev.Rawcode]

			fmt.Printf("\nKey down: %s %d\n", key.Name, ev.Rawcode)
			//	fmt.Printf("-----")
			//	fmt.Printf("key down: rawcode=%d rawcode=0x%x keycode=%d keycode=0x%x keychar=%d keychar=0x%x\n\n",
			//		ev.Rawcode, ev.Rawcode, ev.Keycode, ev.Keycode, ev.Keychar, ev.Keychar)
		} else if ev.Kind == hook.KeyHold {
			key := controller.Raw2Keycode[ev.Rawcode]

			fmt.Printf("\nKey hold: %s %d\n", key.Name, ev.Rawcode)
			//	fmt.Printf("key hold: rawcode=%d rawcode=0x%x keycode=%d keycode=0x%x keychar=%d keychar=0x%x\n\n",
			//		ev.Rawcode, ev.Rawcode, ev.Keycode, ev.Keycode, ev.Keychar, ev.Keychar)
		}
	}
}
