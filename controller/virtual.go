package controller

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
	stick          [2][2]float32
	stick_pressed  [2]bool
	dpad_pressed   [4]bool
	control        [3]bool
	pitch, yaw     float32
}

func B2i(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

func (state *ControllerState) MoveStick(which int, axis int, value float32) {
	state.stick[which][axis] = value
}

func (state *ControllerState) ResetStick(which int, axis int, value float32) {
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

func (state *ControllerState) GetButtonMask() uint8 {
	return ((B2i(state.button_pressed[Y_BUTTON]) << 7) |
		(B2i(state.button_pressed[B_BUTTON]) << 6) |
		(B2i(state.button_pressed[A_BUTTON]) << 5) |
		(B2i(state.button_pressed[X_BUTTON]) << 4) |
		(B2i(state.button_pressed[R_BUTTON]) << 3) |
		(B2i(state.button_pressed[L_BUTTON]) << 2) |
		(B2i(state.button_pressed[ZR_BUTTON]) << 1) |
		(B2i(state.button_pressed[ZL_BUTTON]) << 0))
}

func (state *ControllerState) GetDPadMask() uint8 {
	return ((B2i(state.dpad_pressed[LEFT_DPAD]) << 7) |
		(B2i(state.dpad_pressed[DOWN_DPAD]) << 6) |
		(B2i(state.dpad_pressed[RIGHT_DPAD]) << 5) |
		(B2i(state.dpad_pressed[UP_DPAD]) << 4) |
		(B2i(state.stick_pressed[R_STICK]) << 2) |
		(B2i(state.stick_pressed[L_STICK]) << 1))
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
