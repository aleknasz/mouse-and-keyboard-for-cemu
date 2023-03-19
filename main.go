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
	var prevX, prevY int16 = 0, 0
	var sensitivity float32 = 25.0
	// var mouseSwitch bool = false
	var ignoreMouseMove bool = true
	for ev := range chanHook {

		if ev.Kind == hook.MouseMove || ev.Kind == hook.MouseDrag {

			// y, x := ev.Y, ev.X

			// if prevX == -1 {
			// 	prevX = ev.X
			// }

			// if prevY == -1 {
			// 	prevY = ev.Y
			// }

			var yaw = prevX - ev.X
			var pitch = prevY - ev.Y

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

			var gyro = controller.Vector3{0.0, sensitivity * float32(pitch), sensitivity * -float32(yaw)}

			// if mouseSwitch {
			// 	sx, sy := controller.GetScreenSize()
			// 	controller.MoveMouse(sx/2, sy/2)
			// 	//log.Printf("Mouse event: %v\n", gyro)
			// }

			if webSocketClient != nil {
				webSocketClient.WriteJSON(controller.Vector3{0.0,
					float32(ev.X), float32(ev.Y)})
			}

			//MouseEvent(int(yaw), int(pitch), ignoreMouseMove)

			if !ignoreMouseMove {
				userController.Mouse2Joystick(int(ev.X), int(ev.Y))
			}
			// x_axis := float32(yaw)
			// y_axis := float32(pitch)

			if userController.IsButtonPressed(controller.Y_BUTTON) {
				Report(udpServer, uint64(time.Now().UnixMicro()), controller.ZeroVector3, gyro)
			} else {
				// userController.MoveStick(controller.R_STICK, controller.X_AXIS, x_axis)
				// userController.MoveStick(controller.R_STICK, controller.Y_AXIS, y_axis)

				// log.Printf("Mouse move: %f %f\n", x_axis, y_axis)
				Report(udpServer, uint64(time.Now().UnixMicro()), controller.ZeroVector3, controller.ZeroVector3)
			}

			//log.Printf("\nMouse move: %d %d\n", x_axis, y_axis)

		} else if ev.Kind == hook.KeyUp {

			whenHappened(ev, hook.KeyUp, func() {
				userController.PressStick(controller.R_STICK, false)
			}, controller.DIGIT_5)

			whenHappened(ev, hook.KeyUp, func() {
				userController.PressStick(controller.L_STICK, false)
			}, controller.Command)

			whenHappened(ev, hook.KeyUp, func() {
				userController.PressButton(controller.X_BUTTON, false)
			}, controller.Space)

			whenHappened(ev, hook.KeyUp, func() {
				userController.MoveStick(controller.L_STICK, controller.X_AXIS, 0.0)
			}, controller.CHAR_A)

			whenHappened(ev, hook.KeyUp, func() {
				userController.MoveStick(controller.L_STICK, controller.Y_AXIS, 0.0)
			}, controller.CHAR_S)

			whenHappened(ev, hook.KeyUp, func() {
				userController.MoveStick(controller.L_STICK, controller.X_AXIS, 0.0)
			}, controller.CHAR_D)

			whenHappened(ev, hook.KeyUp, func() {
				userController.MoveStick(controller.L_STICK, controller.Y_AXIS, 0.0)
			}, controller.CHAR_W)

			whenHappened(ev, hook.KeyUp, func() {
				userController.PressDPad(controller.LEFT_DPAD, false)
			}, controller.LeftArrow, controller.DIGIT_1)

			whenHappened(ev, hook.KeyUp, func() {
				userController.PressDPad(controller.DOWN_DPAD, false)
			}, controller.DownArrow, controller.DIGIT_4)

			whenHappened(ev, hook.KeyUp, func() {
				userController.PressDPad(controller.RIGHT_DPAD, false)
			}, controller.RightArrow, controller.DIGIT_3)

			whenHappened(ev, hook.KeyUp, func() {
				userController.PressDPad(controller.UP_DPAD, false)
			}, controller.UpArrow, controller.DIGIT_2)

			whenHappened(ev, hook.KeyUp, func() {
				userController.PressButton(controller.A_BUTTON, false)
			}, controller.Return, controller.CHAR_F)

			whenHappened(ev, hook.KeyUp, func() {
				userController.PressButton(controller.B_BUTTON, false)
			}, controller.Escape, controller.Delete, controller.Shift)

			whenHappened(ev, hook.KeyUp, func() {
				userController.PressButton(controller.L_BUTTON, false)
			}, controller.CHAR_Q)

			whenHappened(ev, hook.KeyUp, func() {
				userController.PressButton(controller.R_BUTTON, false)
			}, controller.CHAR_E)

			whenHappened(ev, hook.KeyUp, func() {
				ignoreMouseMove = !ignoreMouseMove
				log.Printf("Ignore mouse=%d\n", ignoreMouseMove)
				if !ignoreMouseMove {
					userController.SetupMouseZone(false)
				}
			}, controller.ISO_Section)

			Report(udpServer, uint64(time.Now().UnixMicro()), controller.ZeroVector3, controller.ZeroVector3)

			key := controller.Raw2Keycode[ev.Rawcode]
			log.Printf("\nKey Up: %s %d\n", key.Name, ev.Rawcode)

		} else if ev.Kind == hook.KeyHold {
			// key := controller.Raw2Keycode[ev.Rawcode]

			whenHappened(ev, hook.KeyHold, func() {
				userController.PressStick(controller.R_STICK, true)
			}, controller.DIGIT_5)

			whenHappened(ev, hook.KeyHold, func() {
				userController.PressStick(controller.L_STICK, true)
			}, controller.Command)

			whenHappened(ev, hook.KeyHold, func() {
				userController.PressButton(controller.X_BUTTON, true)
			}, controller.Space)

			whenHappened(ev, hook.KeyHold, func() {
				userController.MoveStick(controller.L_STICK, controller.X_AXIS, -1.0)
			}, controller.CHAR_A)

			whenHappened(ev, hook.KeyHold, func() {
				userController.MoveStick(controller.L_STICK, controller.Y_AXIS, -1.0)
			}, controller.CHAR_S)

			whenHappened(ev, hook.KeyHold, func() {
				userController.MoveStick(controller.L_STICK, controller.X_AXIS, 1.0)
			}, controller.CHAR_D)

			whenHappened(ev, hook.KeyHold, func() {
				userController.MoveStick(controller.L_STICK, controller.Y_AXIS, 1.0)
			}, controller.CHAR_W)

			whenHappened(ev, hook.KeyHold, func() {
				userController.PressDPad(controller.LEFT_DPAD, true)
			}, controller.LeftArrow, controller.DIGIT_1)

			whenHappened(ev, hook.KeyHold, func() {
				userController.PressDPad(controller.DOWN_DPAD, true)
			}, controller.DownArrow, controller.DIGIT_4)

			whenHappened(ev, hook.KeyHold, func() {
				userController.PressDPad(controller.RIGHT_DPAD, true)
			}, controller.RightArrow, controller.DIGIT_3)

			whenHappened(ev, hook.KeyHold, func() {
				userController.PressDPad(controller.UP_DPAD, true)
			}, controller.UpArrow, controller.DIGIT_2)

			whenHappened(ev, hook.KeyHold, func() {
				userController.PressButton(controller.A_BUTTON, true)
			}, controller.Return, controller.CHAR_F)

			whenHappened(ev, hook.KeyHold, func() {
				userController.PressButton(controller.B_BUTTON, true)
			}, controller.Escape, controller.Delete, controller.Shift)

			whenHappened(ev, hook.KeyHold, func() {
				userController.PressButton(controller.L_BUTTON, true)
			}, controller.CHAR_Q)

			whenHappened(ev, hook.KeyHold, func() {
				userController.PressButton(controller.R_BUTTON, true)
			}, controller.CHAR_E)

			Report(udpServer, uint64(time.Now().UnixMicro()), controller.ZeroVector3, controller.ZeroVector3)

			key := controller.Raw2Keycode[ev.Rawcode]
			log.Printf("\nKey hold: %s %d\n", key.Name, ev.Rawcode)

			// log.Printf("\nKey hold: %d %d\n", userController.GetDPadMask(), userController.IsDPadPressed(controller.LEFT_DPAD))
			//	log.Printf("key hold: rawcode=%d rawcode=0x%x keycode=%d keycode=0x%x keychar=%d keychar=0x%x\n\n",
			//		ev.Rawcode, ev.Rawcode, ev.Keycode, ev.Keycode, ev.Keychar, ev.Keychar)

		} else if ev.Kind == hook.MouseUp || ev.Kind == hook.MouseDown {
			button := ev.Button
			if button == 1 {
				userController.PressButton(controller.Y_BUTTON, false)
				// Report(udpServer, uint64(time.Now().UnixMicro()), controller.ZeroVector3, controller.ZeroVector3)
			} else if button == 2 {
				userController.PressButton(controller.ZR_BUTTON, false)

			}

			Report(udpServer, uint64(time.Now().UnixMicro()), controller.ZeroVector3, controller.ZeroVector3)

			log.Printf("\nMouse up: %d ignore=%d\n", button, ignoreMouseMove)
		} else if ev.Kind == hook.MouseHold {
			button := ev.Button
			if button == 1 {
				userController.PressButton(controller.Y_BUTTON, true)
				// Report(udpServer, uint64(time.Now().UnixMicro()), controller.ZeroVector3, controller.ZeroVector3)
			} else if button == 2 {
				userController.PressButton(controller.ZR_BUTTON, true)

			}

			Report(udpServer, uint64(time.Now().UnixMicro()), controller.ZeroVector3, controller.ZeroVector3)

			log.Printf("\nMouse hold: %d ignore=%d\n", button, ignoreMouseMove)
		} else if ev.Kind == hook.MouseWheel {
			// if ev.Rotation > 0 {
			userController.MoveStick(controller.R_STICK, controller.X_AXIS, float64(ev.Rotation))
			// } else if ev.Rotation < 0 {
			// userController.MoveStick(controller.R_STICK, controller.X_AXIS, 1.0)
			// }
			log.Printf("\nMouse wheel: %v %v %v\n", ev.Amount, ev.Rotation, ev.Direction)

			Report(udpServer, uint64(time.Now().UnixMicro()), controller.ZeroVector3, controller.ZeroVector3)

			go func() {
				userController.MoveStick(controller.R_STICK, controller.X_AXIS, 0.0)
				Report(udpServer, uint64(time.Now().UnixMicro()), controller.ZeroVector3, controller.ZeroVector3)
			}()
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
