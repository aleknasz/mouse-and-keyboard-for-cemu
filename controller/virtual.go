package controller

import "math"

const (
	A_BUTTON  = 0
	B_BUTTON  = 1
	X_BUTTON  = 2
	Y_BUTTON  = 3
	L_BUTTON  = 4
	ZL_BUTTON = 5
	R_BUTTON  = 6
	ZR_BUTTON = 7
)

const (
	L_STICK = 0
	R_STICK = 1
)

const (
	X_AXIS = 0
	Y_AXIS = 1
)

const (
	UP_DPAD    = 0
	RIGHT_DPAD = 1
	DOWN_DPAD  = 2
	LEFT_DPAD  = 3
)

const (
	PLUS_CONTROL  = 0
	MINUS_CONTROL = 1
	HOME_CONTROL  = 2
)

type ControllerState struct {
	button_pressed [8]bool
	stick          [2][2]float64
	stick_pressed  [2]bool
	dpad_pressed   [4]bool
	control        [3]bool
	pitch, yaw     float64

	alt_xSen, alt_ySen int

	useX, useY, xZero, yZero int

	nnp                  float64
	r                    float64
	k                    float64
	dr                   float64
	invertedX, invertedY bool
	alreadyDown          bool

	gameX, gameY   int
	gameW, gameH   int
	OX, OY         int
	pmX, pmY       float64
	snapToFullTilt float64
}

func b2i(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

func (state *ControllerState) MoveStick(which int, axis int, value float64) {
	state.stick[which][axis] = value
}

func (state *ControllerState) ResetStick(which int, axis int, value float64) {
	state.stick[which][axis] = 0.0
}

func (state *ControllerState) GetStickMask(which int, axis int) byte {
	if state.stick[which][axis] < 0.0 {
		return 0x00
	}
	if state.stick[which][axis] > 0.0 {
		return 0xFF
	}
	return 0x80
}

func (state *ControllerState) GetStickValue(which int, axis int) byte {
	if state.stick[which][axis] == 0.0 {
		return 0x80
	}
	// to_return := byte((state.stick[which][axis] + 1.0) * 128)
	// return to_return
	return byte(math.Min(math.Max(128.0+float64(state.stick[which][axis]*128.0), 0.0), 255.0))
}

func (state *ControllerState) GetButtonMask() uint8 {
	return ((b2i(state.button_pressed[Y_BUTTON]) << 7) |
		(b2i(state.button_pressed[B_BUTTON]) << 6) |
		(b2i(state.button_pressed[A_BUTTON]) << 5) |
		(b2i(state.button_pressed[X_BUTTON]) << 4) |
		(b2i(state.button_pressed[R_BUTTON]) << 3) |
		(b2i(state.button_pressed[L_BUTTON]) << 2) |
		(b2i(state.button_pressed[ZR_BUTTON]) << 1) |
		(b2i(state.button_pressed[ZL_BUTTON]) << 0))
}

func (state *ControllerState) GetDPadMask() uint8 {
	return ((b2i(state.dpad_pressed[LEFT_DPAD]) << 7) |
		(b2i(state.dpad_pressed[DOWN_DPAD]) << 6) |
		(b2i(state.dpad_pressed[RIGHT_DPAD]) << 5) |
		(b2i(state.dpad_pressed[UP_DPAD]) << 4) |
		(b2i(state.stick_pressed[R_STICK]) << 2) |
		(b2i(state.stick_pressed[L_STICK]) << 1))
}

func (state *ControllerState) PressButton(index int, pressed bool) {
	state.button_pressed[index] = pressed
}

func (state *ControllerState) IsButtonPressed(index int) bool {
	return state.button_pressed[index]
}

func (state *ControllerState) PressStick(index int, pressed bool) {
	state.stick_pressed[index] = pressed
}

func (state *ControllerState) IsStickPressed(index int) bool {
	return state.stick_pressed[index]
}

func (state *ControllerState) PressDPad(index int, pressed bool) {
	state.dpad_pressed[index] = pressed
}

func (state *ControllerState) IsDPadPressed(index int) bool {
	return state.dpad_pressed[index]
}

func (state *ControllerState) PressControl(index int, pressed bool) {
	state.control[index] = pressed
}

func (state *ControllerState) IsControlPressed(index int) bool {
	return state.control[index]
}

func (state *ControllerState) Mouse2Joystick(X int, Y int) {
	X -= state.OX // Move to controller coord system.
	Y -= state.OY
	var RR = math.Sqrt(float64(X*X + Y*Y))
	if RR > state.r { // Check If outside controller circle.
		X = int(math.Round(float64(X) * (state.r - state.dr) / RR))
		Y = int(math.Round(float64(Y) * (state.r - state.dr) / RR))
		RR = math.Sqrt(float64(X*X + Y*Y))
		// controller.MoveMouse(X+OX, Y+OY) // Calculate point on controller circle, move back to screen/window coords, and move mouse.
	}

	// Calculate angle
	phi := getAngle(X, Y)

	if RR > state.k*state.r && !state.alreadyDown { // Check If outside inner circle/deadzone.
		state.action(phi, math.Pow(((RR-state.k*state.r)/(state.r-state.k*state.r)), state.nnp)) // nnp is a non-linearity parameter.
	} else {
		state.setStick(0, 0) // Stick in equllibrium.
	}
	// controller.MoveMouse(OX, OY)
}

func (state *ControllerState) action(phi float64, tilt float64) {

	if tilt > 1.0 {
		tilt = 1.0
	}

	if state.snapToFullTilt != -1.0 {
		if 1-tilt <= state.snapToFullTilt {
			tilt = 1.0
		}
	}
	var lb, ub float64
	// Two cases with forward+right
	// Tilt is forward and slightly right.
	lb = 3 * math.Pi / 2.0 // lb is lower bound
	ub = 7 * math.Pi / 4.0 // ub is upper bound
	if phi >= lb && phi <= ub {
		x := state.pmX * tilt * scale(phi, ub, lb)
		y := state.pmY * tilt
		state.setStick(x, y)
		return
	}
	// Tilt is slightly forward and right.
	lb = 7 * math.Pi / 4 // lb is lower bound
	ub = 2 * math.Pi     // ub is upper bound
	if phi >= lb && phi <= ub {
		x := state.pmX * tilt
		y := state.pmY * tilt * scale(phi, lb, ub)
		state.setStick(x, y)
		return
	}

	// Two cases with right+downward
	// Tilt is right and slightly downward.
	lb = 0           // lb is lower bound
	ub = math.Pi / 4 // ub is upper bound
	if phi >= lb && phi <= ub {
		x := state.pmX * tilt
		y := -state.pmY * tilt * scale(phi, ub, lb)
		state.setStick(x, y)
		return
	}
	// Tilt is downward and slightly right.
	lb = math.Pi / 4 // lb is lower bound
	ub = math.Pi / 2 // ub is upper bound
	if phi >= lb && phi <= ub {
		x := state.pmX * tilt * scale(phi, lb, ub)
		y := -state.pmY * tilt
		state.setStick(x, y)
		return
	}

	// Two cases with downward+left
	// Tilt is downward and slightly left.
	lb = math.Pi / 2     // lb is lower bound
	ub = 3 * math.Pi / 4 // ub is upper bound
	if phi >= lb && phi <= ub {
		x := -state.pmX * tilt * scale(phi, ub, lb)
		y := -state.pmY * tilt
		state.setStick(x, y)
		return
	}
	// Tilt is left and slightly downward.
	lb = 3 * math.Pi / 4 // lb is lower bound
	ub = math.Pi         // ub is upper bound
	if phi >= lb && phi <= ub {
		x := -state.pmX * tilt
		y := -state.pmY * tilt * scale(phi, lb, ub)
		state.setStick(x, y)
		return
	}

	// Two cases with forward+left
	// Tilt is left and slightly forward.
	lb = math.Pi         // lb is lower bound
	ub = 5 * math.Pi / 4 // ub is upper bound
	if phi >= lb && phi <= ub {
		x := -state.pmX * tilt
		y := state.pmY * tilt * scale(phi, ub, lb)
		state.setStick(x, y)
		return
	}
	// Tilt is forward and slightly left.
	lb = 5 * math.Pi / 4 // lb is lower bound
	ub = 3 * math.Pi / 2 // ub is upper bound
	if phi >= lb && phi <= ub {
		x := -state.pmX * tilt * scale(phi, lb, ub)
		y := state.pmY * tilt
		state.setStick(x, y)
		return
	}
	// This should not happen
	state.setStick(0, 0)
}

func (state *ControllerState) setStick(x float64, y float64) {
	// x,y are from range (-1,1) which is mapped later to (0,255)
	//log.Printf("Setting stick to x=%f, y=%f", x, y)
	state.MoveStick(R_STICK, X_AXIS, x)
	state.MoveStick(R_STICK, Y_AXIS, y)
}

func scale(phi float64, lb float64, ub float64) float64 {
	// let phi->f(phi) then, f(ub)=0 and f(lb)=1
	return (phi - ub) / (lb - ub)
}

func getAngle(x int, y int) float64 {
	if x == 0 {
		var p float64
		if y > 0 {
			p = math.Pi
		} else {
			p = 0
		}
		return 3*math.Pi/2.0 - p
	}
	phi := math.Atan(float64(y) / float64(x))
	if x < 0 && y > 0 {
		return phi + math.Pi
	}
	if x < 0 && y <= 0 {
		return phi + math.Pi
	}
	if x > 0 && y < 0 {
		return phi + 2*math.Pi
	}
	return phi
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (state *ControllerState) SetupMouseZone(center bool) {

	state.alt_xSen = 400
	state.alt_ySen = 280

	state.nnp = 0.8
	state.r = 60
	state.k = 0.02
	state.dr = 0
	state.invertedX = false
	state.invertedY = true
	state.alreadyDown = false

	state.gameX, state.gameY = 0, 0
	state.gameW, state.gameH = 0, 0
	state.OX, state.OY = 0, 0
	state.snapToFullTilt = 0.005

	if state.invertedX {
		state.pmX = -1
	} else {
		state.pmX = 1
	}

	if state.invertedY {
		state.pmY = -1
	} else {
		state.pmY = 1
	}

	state.gameW, state.gameH = GetScreenSize()

	if center {
		state.OX = state.gameX + state.gameW/2
		state.OY = state.gameY + state.gameH/2
		MoveMouse(state.OX, state.OY)
	} else {
		state.OX, state.OY = GetMousePos()
	}

}

func (state *ControllerState) MouseEvent(x int, y int, ignoreMouseEvent bool) {
	intv := 1

	if (x < 0 && state.useX > 0) || (x > 0 && state.useX < 0) {
		state.useX = 0
	}
	if (y < 0 && state.useY > 0) || (y > 0 && state.useY < 0) {
		state.useY = 0
	}
	if x != 0 && y != 0 {
		intv = 4
	}
	if x == 0 {
		state.xZero++
	}
	if state.xZero > 2 {
		state.useX = 0
		state.xZero = 0
	}
	if x > 0 {
		state.useX += intv
	} else {
		state.useX -= intv
	}
	if y == 0 {
		state.yZero++
	}
	if state.yZero > 2 {
		state.useY = 0
		state.yZero = 0
	}
	if y > 0 {
		state.useY += intv
	} else {
		state.useY -= intv
	}
	if abs(state.useX) > state.alt_xSen {
		state.useX = state.useX / abs(state.useX) * state.alt_xSen
	} else if abs(x) != 0 && abs(state.useX) < state.alt_xSen/6.0 {
		state.useX = state.useX / abs(state.useX) * state.alt_xSen / 6.0
	}

	if abs(state.useY) > state.alt_ySen {
		state.useY = state.useY / abs(state.useY) * state.alt_ySen
	} else if abs(y) != 0 && abs(state.useY) < state.alt_ySen/6.0 {
		state.useY = state.useY / abs(state.useY) * state.alt_ySen / 6.0
	}

	if ignoreMouseEvent {
		state.MoveStick(R_STICK, X_AXIS, 0)
		state.MoveStick(R_STICK, Y_AXIS, 0)
	} else {
		state.MoveStick(R_STICK, X_AXIS, -float64(state.useX)/float64(state.alt_xSen))
		state.MoveStick(R_STICK, Y_AXIS, float64(state.useY)/float64(state.alt_ySen))
	}

}
