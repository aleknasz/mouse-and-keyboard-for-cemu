package main

import (
	"encoding/json"
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

var udpServer net.PacketConn

var frameCount = 0
var fps = int64(60)
var fpsInterval = 1000 / fps
var then = time.Now().UnixMilli()
var startTime = then

// Just setup key events listener and udp server for DSU protocol
// Also there is some legacy logic for little web site serving and motion control receiving from android device
func main() {

	//var outBuffer = make([]byte, 100)
	//now := 1677329375368994 //time.Now().UnixMicro()
	// time.Now().UnixMilli()
	//binary.LittleEndian.PutUint32(outBuffer, math.Float32bits(-123.45))
	//binary.LittleEndian.PutUint64(outBuffer, uint64(now))
	//log.Printf("Wrote %s %d %d\n", hex.EncodeToString(outBuffer), time.Now().UnixMicro(), now)
	rand.Seed(time.Now().UnixNano())

	us, err := net.ListenPacket("udp", ":26760")
	if err != nil {
		panic(err)
	}
	udpServer = us
	defer udpServer.Close()

	// This was previously used for getting motion control from Android Phone
	// Recently was used to send debug information back to Android Phone
	// But now, we are enough mature to disable that at all since Phone is not needed anymore (for motion or debugging)
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "index.html")
		})
		http.HandleFunc("/ws", webSocketEndpoint)
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			panic(err)
		}
	}()

	chanHook := hook.Start()
	defer hook.End()

	go keyEventsLoop(udpServer, chanHook)

	receiveUDPLoop(udpServer)

}

// This processes all the request coming from game
func receiveUDPLoop(udpServer net.PacketConn) {
	data := make([]byte, 512)
	var prot controller.DSUProtocol
	for {

		readLength, addr, err := udpServer.ReadFrom(data)
		if err != nil {
			continue
		}

		if readLength == 0 {
			log.Printf("Empty receive\n")
			continue
		}

		prot.ReadRequest(data)

		if prot.MessageType == controller.DSUC_VersionReq {
			log.Printf("Version request, ignoring\n")
		} else if prot.MessageType == controller.DSUC_ListPorts {
			log.Printf("Get state of %d port(s)\n", prot.NumOfPadRequests)
			for i := 0; i < prot.NumOfPadRequests; i += 1 {
				requestIndex := prot.RequestIndex[i]
				if requestIndex != 0 {
					continue
				}

				response := prot.CreateListPortsResponse()

				go udpServer.WriteTo(response, addr)
			}
		} else if prot.MessageType == controller.DSUC_PadDataReq {
			//log.Printf("Pad data request for %s with flags %d and id %d\n", macToRegister, flags,
			//	idToRRegister)
			if (prot.Flags == 0 || (prot.IdToRRegister == 0 && (prot.Flags&0x01) != 0)) ||
				((prot.MacToRegister == "00:00:00:00:00:ff") && (prot.Flags&0x02) != 0) {
				lastRequestAt = time.Now().UnixMilli()
				if connectedClient == nil {
					log.Printf("Game connected from %v at %d\n", addr, lastRequestAt)
				}
				connectedClient = addr
			}
		}
	}
}

// Send back status of our controller
func Report(udpServer net.PacketConn, motionTimestamp uint64, accelerometer controller.Vector3, gyro controller.Vector3) {
	client := connectedClient
	if client == nil {
		return
	}

	if time.Now().UnixMilli()-lastRequestAt > clientTimeoutLimit {
		log.Printf("Game timeout %s, disconnecting\n", client)
		connectedClient = nil
		return
	}

	//log.Printf("Unmarshalled %v\n", report)

	//if report.Ts == "" {
	//	return
	//}

	//log.Printf("Send: %s at %d\n",
	//	hex.EncodeToString(outBuffer), motionTimestamp)

	//log.Printf("Send package to %s at %d\n", client, lastRequestAt)

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

// Some key or button has been pressed or mouse has been moved
func keyEventsLoop(udpServer net.PacketConn, chanHook <-chan hook.Event) {
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
			//	log.Printf("Skip %d\n", packetCounter)
			//	Report(udpServer, 0, 0, 0)
			//} else {
			//	log.Printf("Pos: %d %d %d\n", yaw, pitch, packetCounter)
			//	Report(udpServer, yaw, pitch, time.Now().UnixMilli())
			//}

			var gyro = controller.Vector3{0.0, sensitivity * pitch, sensitivity * -yaw}

			if mouseSwitch {
				sx, sy := robot.GetScreenSize()
				robot.Move(sx/2, sy/2)
				//log.Printf("Mouse event: %v\n", gyro)
			}

			if webSocketClient != nil {
				webSocketClient.WriteJSON(controller.Vector3{0.0,
					float32(ev.X), float32(ev.Y)})
			}

			Report(udpServer, uint64(time.Now().UnixMicro()), controller.ZeroVector3, gyro)

			// log.Printf("\nMouse move: %d %d\n", x, y)

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

			// log.Printf("\nKey up: %x\n", userController.GetDPadMask())

			log.Printf("\nKey Up: %s %d\n", key.Name, ev.Rawcode)

			//	log.Printf("key up: rawcode=%d rawcode=0x%x keycode=%d keycode=0x%x keychar=%d keychar=0x%x\n\n",
			//		ev.Rawcode, ev.Rawcode, ev.Keycode, ev.Keycode, ev.Keychar, ev.Keychar)
			//} else if ev.Kind == hook.KeyDown {
			//	key := controller.Raw2Keycode[ev.Rawcode]
			//
			//	log.Printf("\nKey down: %s %d\n", key.Name, ev.Rawcode)
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

			// log.Printf("\nKey hold: %d %d\n", userController.GetDPadMask(), userController.IsDPadPressed(controller.LEFT_DPAD))
			//	log.Printf("key hold: rawcode=%d rawcode=0x%x keycode=%d keycode=0x%x keychar=%d keychar=0x%x\n\n",
			//		ev.Rawcode, ev.Rawcode, ev.Keycode, ev.Keycode, ev.Keychar, ev.Keychar)
		} else if ev.Kind == hook.MouseDown {
			// button := ev.Button
			// log.Printf("\nMouse down: %d\n", button)
		} else if ev.Kind == hook.MouseUp {
			// button := ev.Button
			// log.Printf("\nMouse up: %d\n", button)
		} else if ev.Kind == hook.MouseHold {
			// button := ev.Button
			// log.Printf("\nMouse hold: %d\n", button)
		}
		//else if ev.Kind == hook.MouseMove {
		//	y, x := ev.Y, ev.X
		//	log.Printf("\nMouse move: %d %d\n", x, y)
		//}
		//else if ev.Kind == hook.MouseDrag {
		//	button := ev.Button
		//	log.Printf("\nMouse drag: %d\n", button)
		//}
	}
}

// TODO: All below to be removed

type PhoneReport struct {
	Ts   string             `json:"ts"`
	Gyro controller.Vector3 `json:"gyro"`
}

func webSocketEndpoint(w http.ResponseWriter, r *http.Request) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		panic(err)
	}
	log.Printf("Web client connected\n")
	webSocketClient = ws
	// webSocketReader(ws)
}

func webSocketReader(conn *websocket.Conn) {
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
			log.Printf("Too fast %d, expected %d\n", elapsed, fpsInterval)
			continue
		}

		//log.Printf("OK %d, expected %d\n", elapsed, fpsInterval)

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
			log.Printf("Error %v\n", err)
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
