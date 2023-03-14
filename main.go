package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"mouse-and-keyboard-for-cemu/controller"
	"net"
	"net/http"
	"strconv"
	"time"

	robot "github.com/go-vgo/robotgo"
	websocket "github.com/gorilla/websocket"
	hook "github.com/robotn/gohook"
)

var userController controller.ControllerState

const clientTimeoutLimit = 30000

var lastRequestAt = time.Now().UnixMilli()
var connectedClient net.Addr = nil

var webSocketClient *websocket.Conn = nil

// var report PhoneReport
var udpServer net.PacketConn

var frameCount = 0
var fps = int64(60)
var fpsInterval = 1000 / fps
var then = time.Now().UnixMilli()
var startTime = then

type PhoneReport struct {
	Ts   string             `json:"ts"`
	Gyro controller.Vector3 `json:"gyro"`
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
		Report(udpServer, motionTimestamp, controller.ZeroVector3, report.Gyro)
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
	var prot controller.DSUProtocol
	for {

		readLength, addr, err := udpServer.ReadFrom(data)
		if err != nil {
			continue
		}

		// length := len(data)

		//fmt.Printf("Received %d bytes, elapsed %v, expected %v\n", length)

		if readLength == 0 {
			fmt.Printf("Empty receive\n")
			continue
		}

		prot.ReadRequest(data)

		// index := 4                                              // for now skip header
		// index += 2                                              // skip protocol
		// index += 2                                              // skip packetSize
		// index += 4                                              // skip crc
		// index += 4                                              // skip client id
		// messageType := binary.LittleEndian.Uint32(data[index:]) // care only about message type
		// index += 4

		if prot.MessageType == controller.DSUC_VersionReq {
			//fmt.Printf("Version request\n")
		} else if prot.MessageType == controller.DSUC_ListPorts {
			fmt.Printf("List ports\n")

			// numOfPadRequests := int(binary.LittleEndian.Uint32(data[index:]))
			// index += 4
			for i := 0; i < prot.NumOfPadRequests; i += 1 {
				requestIndex := prot.RequestIndex[i]
				//data[index+i]
				if requestIndex != 0 {
					continue
				}

				response := prot.CreateListPortsResponse()

				go udpServer.WriteTo(response, addr)
			}
		} else if prot.MessageType == controller.DSUC_PadDataReq {

			//fmt.Printf("Pad data request for %s with flags %d and id %d\n", macToRegister, flags,
			//	idToRRegister)
			if (prot.Flags == 0 || (prot.IdToRRegister == 0 && (prot.Flags&0x01) != 0)) ||
				((prot.MacToRegister == "00:00:00:00:00:ff") && (prot.Flags&0x02) != 0) {
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

func Report(udpServer net.PacketConn, motionTimestamp uint64, accelerometer controller.Vector3, gyro controller.Vector3) {
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

	//fmt.Printf("Send: %s at %d\n",
	//	hex.EncodeToString(outBuffer), motionTimestamp)

	//fmt.Printf("Send package to %s at %d\n", client, lastRequestAt)

	var prot controller.DSUProtocol

	response := prot.CreateControllerResponse(&userController, motionTimestamp, gyro)

	go udpServer.WriteTo(response, client)
}

// Study what was pressed and trigger action if it is in our scope of interest
func whenHappened(event hook.Event, eventType uint8, action func(), expectations ...controller.KeyCode) bool {
	if event.Kind == eventType {
		for _, e := range expectations {
			if e.RawCode == event.Rawcode {
				action()
				return true
			}
		}
	}
	return false
}

func captureEvents(udpServer net.PacketConn, chanHook <-chan hook.Event) {
	var prevX, prevY int16 = -1, -1
	var sensitivity float32 = 25.0
	var mouseSwitch bool = false
	for ev := range chanHook {

		if ev.Kind == hook.MouseMove || ev.Kind == hook.MouseDrag {

			// y, x := ev.Y, ev.X

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

			var gyro = controller.Vector3{0.0, sensitivity * pitch, sensitivity * -yaw}

			if mouseSwitch {
				sx, sy := robot.GetScreenSize()
				robot.Move(sx/2, sy/2)
				//fmt.Printf("Mouse event: %v\n", gyro)
			}

			if webSocketClient != nil {
				webSocketClient.WriteJSON(controller.Vector3{0.0,
					float32(ev.X), float32(ev.Y)})
			}

			Report(udpServer, uint64(time.Now().UnixMicro()), controller.ZeroVector3, gyro)

			// fmt.Printf("\nMouse move: %d %d\n", x, y)

		} else if ev.Kind == hook.KeyUp {

			whenHappened(ev, hook.KeyUp, func() {
				userController.PressDPad(controller.LEFT_DPAD, false)
			}, controller.CHAR_A, controller.LeftArrow)

			whenHappened(ev, hook.KeyUp, func() {
				userController.PressDPad(controller.DOWN_DPAD, false)
			}, controller.CHAR_S, controller.DownArrow)

			whenHappened(ev, hook.KeyUp, func() {
				userController.PressDPad(controller.RIGHT_DPAD, false)
			}, controller.CHAR_D, controller.RightArrow)

			whenHappened(ev, hook.KeyUp, func() {
				userController.PressDPad(controller.UP_DPAD, false)
			}, controller.CHAR_W, controller.UpArrow)

			whenHappened(ev, hook.KeyUp, func() {
				userController.PressButton(controller.A_BUTTON, false)
			}, controller.Return)

			whenHappened(ev, hook.KeyUp, func() {
				userController.PressButton(controller.B_BUTTON, false)
			}, controller.ISO_Section, controller.Escape, controller.Delete)

			key := controller.Raw2Keycode[ev.Rawcode]

			// if ev.Rawcode == controller.CHAR_A.RawCode || ev.Rawcode == controller.LeftArrow.RawCode {
			// 	userController.PressDPad(controller.LEFT_DPAD, false)
			// } else if ev.Rawcode == controller.CHAR_S.RawCode || ev.Rawcode == controller.DownArrow.RawCode {
			// 	userController.PressDPad(controller.DOWN_DPAD, false)
			// } else if ev.Rawcode == controller.CHAR_D.RawCode || ev.Rawcode == controller.RightArrow.RawCode {
			// 	userController.PressDPad(controller.RIGHT_DPAD, false)
			// } else if ev.Rawcode == controller.CHAR_W.RawCode || ev.Rawcode == controller.UpArrow.RawCode {
			// 	userController.PressDPad(controller.UP_DPAD, false)
			// } else if ev.Rawcode == controller.Return.RawCode {
			// 	userController.PressButton(controller.A_BUTTON, false)
			// } else if ev.Rawcode == controller.ISO_Section.RawCode || ev.Rawcode == controller.Escape.RawCode {
			// 	userController.PressButton(controller.B_BUTTON, false)
			// }

			Report(udpServer, uint64(time.Now().UnixMicro()), controller.ZeroVector3, controller.ZeroVector3)

			// fmt.Printf("\nKey up: %x\n", userController.GetDPadMask())

			fmt.Printf("\nKey Up: %s %d\n", key.Name, ev.Rawcode)

			//	fmt.Printf("key up: rawcode=%d rawcode=0x%x keycode=%d keycode=0x%x keychar=%d keychar=0x%x\n\n",
			//		ev.Rawcode, ev.Rawcode, ev.Keycode, ev.Keycode, ev.Keychar, ev.Keychar)
			//} else if ev.Kind == hook.KeyDown {
			//	key := controller.Raw2Keycode[ev.Rawcode]
			//
			//	fmt.Printf("\nKey down: %s %d\n", key.Name, ev.Rawcode)
		} else if ev.Kind == hook.KeyHold {
			// key := controller.Raw2Keycode[ev.Rawcode]

			whenHappened(ev, hook.KeyHold, func() {
				userController.PressDPad(controller.LEFT_DPAD, true)
			}, controller.CHAR_A, controller.LeftArrow)

			whenHappened(ev, hook.KeyHold, func() {
				userController.PressDPad(controller.DOWN_DPAD, true)
			}, controller.CHAR_S, controller.DownArrow)

			whenHappened(ev, hook.KeyHold, func() {
				userController.PressDPad(controller.RIGHT_DPAD, true)
			}, controller.CHAR_D, controller.RightArrow)

			whenHappened(ev, hook.KeyHold, func() {
				userController.PressDPad(controller.UP_DPAD, true)
			}, controller.CHAR_W, controller.UpArrow)

			whenHappened(ev, hook.KeyHold, func() {
				userController.PressButton(controller.A_BUTTON, true)
			}, controller.Return)

			whenHappened(ev, hook.KeyHold, func() {
				userController.PressButton(controller.B_BUTTON, true)
			}, controller.ISO_Section, controller.Escape, controller.Delete)

			// if ev.Rawcode == controller.CHAR_A.RawCode || ev.Rawcode == controller.LeftArrow.RawCode {
			// 	userController.PressDPad(controller.LEFT_DPAD, true)
			// } else if ev.Rawcode == controller.CHAR_S.RawCode || ev.Rawcode == controller.DownArrow.RawCode {
			// 	userController.PressDPad(controller.DOWN_DPAD, true)
			// } else if ev.Rawcode == controller.CHAR_D.RawCode || ev.Rawcode == controller.RightArrow.RawCode {
			// 	userController.PressDPad(controller.RIGHT_DPAD, true)
			// } else if ev.Rawcode == controller.CHAR_W.RawCode || ev.Rawcode == controller.UpArrow.RawCode {
			// 	userController.PressDPad(controller.UP_DPAD, true)
			// } else if ev.Rawcode == controller.Return.RawCode {
			// 	userController.PressButton(controller.A_BUTTON, true)
			// } else if ev.Rawcode == controller.ISO_Section.RawCode || ev.Rawcode == controller.Escape.RawCode {
			// 	userController.PressButton(controller.B_BUTTON, true)
			// }

			Report(udpServer, uint64(time.Now().UnixMicro()), controller.ZeroVector3, controller.ZeroVector3)

			// fmt.Printf("\nKey hold: %d %d\n", userController.GetDPadMask(), userController.IsDPadPressed(controller.LEFT_DPAD))
			//	fmt.Printf("key hold: rawcode=%d rawcode=0x%x keycode=%d keycode=0x%x keychar=%d keychar=0x%x\n\n",
			//		ev.Rawcode, ev.Rawcode, ev.Keycode, ev.Keycode, ev.Keychar, ev.Keychar)
		} else if ev.Kind == hook.MouseDown {
			// button := ev.Button
			// fmt.Printf("\nMouse down: %d\n", button)
		} else if ev.Kind == hook.MouseUp {
			// button := ev.Button
			// fmt.Printf("\nMouse up: %d\n", button)
		} else if ev.Kind == hook.MouseHold {
			// button := ev.Button
			// fmt.Printf("\nMouse hold: %d\n", button)
		}
		//else if ev.Kind == hook.MouseMove {
		//	y, x := ev.Y, ev.X
		//	fmt.Printf("\nMouse move: %d %d\n", x, y)
		//}
		//else if ev.Kind == hook.MouseDrag {
		//	button := ev.Button
		//	fmt.Printf("\nMouse drag: %d\n", button)
		//}
	}
}
