package controller

type KeyCode struct {
	Keycode uint16
	Name    string
}

var (
	CHAR_A          = KeyCode{0, "A"}
	CHAR_S          = KeyCode{1, "S"}
	CHAR_D          = KeyCode{2, "D"}
	CHAR_F          = KeyCode{3, "F"}
	CHAR_H          = KeyCode{4, "H"}
	CHAR_G          = KeyCode{5, "G"}
	CHAR_Z          = KeyCode{6, "Z"}
	CHAR_X          = KeyCode{7, "X"}
	CHAR_C          = KeyCode{8, "C"}
	CHAR_V          = KeyCode{9, "V"}
	ISO_Section     = KeyCode{10, "ISO_Section"}
	CHAR_B          = KeyCode{11, "B"}
	CHAR_Q          = KeyCode{12, "Q"}
	CHAR_W          = KeyCode{13, "W"}
	CHAR_E          = KeyCode{14, "E"}
	CHAR_R          = KeyCode{15, "R"}
	CHAR_Y          = KeyCode{16, "Y"}
	CHAR_T          = KeyCode{17, "T"}
	DIGIT_1         = KeyCode{18, "1"}
	DIGIT_2         = KeyCode{19, "2"}
	DIGIT_3         = KeyCode{20, "3"}
	DIGIT_4         = KeyCode{21, "4"}
	DIGIT_6         = KeyCode{22, "6"}
	DIGIT_5         = KeyCode{23, "5"}
	Equal           = KeyCode{24, "Equal"}
	DIGIT_9         = KeyCode{25, "9"}
	DIGIT_7         = KeyCode{26, "7"}
	Minus           = KeyCode{27, "Minus"}
	DIGIT_8         = KeyCode{28, "8"}
	DIGIT_0         = KeyCode{29, "0"}
	RightBracket    = KeyCode{30, "RightBracket"}
	CHAR_O          = KeyCode{31, "O"}
	CHAR_U          = KeyCode{32, "U"}
	LeftBracket     = KeyCode{33, "LeftBracket"}
	CHAR_I          = KeyCode{34, "I"}
	CHAR_P          = KeyCode{35, "P"}
	Return          = KeyCode{36, "Return"}
	CHAR_L          = KeyCode{37, "L"}
	CHAR_J          = KeyCode{38, "J"}
	Quote           = KeyCode{39, "Quote"}
	CHAR_K          = KeyCode{40, "K"}
	Semicolon       = KeyCode{41, "Semicolon"}
	Backslash       = KeyCode{42, "Backslash"}
	Comma           = KeyCode{43, "Comma"}
	Slash           = KeyCode{44, "Slash"}
	CHAR_N          = KeyCode{45, "N"}
	CHAR_M          = KeyCode{46, "M"}
	Period          = KeyCode{47, "Period"}
	Tab             = KeyCode{48, "Tab"}
	Space           = KeyCode{49, "Space"}
	Grave           = KeyCode{50, "Grave"}
	Delete          = KeyCode{51, "Delete"}
	Escape          = KeyCode{53, "Escape"}
	RightCommand    = KeyCode{54, "RightCommand"}
	Command         = KeyCode{55, "Command"}
	Shift           = KeyCode{56, "Shift"}
	CapsLock        = KeyCode{57, "CapsLock"}
	Option          = KeyCode{58, "Option"}
	Control         = KeyCode{59, "Control"}
	RightShift      = KeyCode{60, "RightShift"}
	RightOption     = KeyCode{61, "RightOption"}
	RightControl    = KeyCode{62, "RightControl"}
	Function        = KeyCode{63, "Function"}
	F17             = KeyCode{64, "F17"}
	KeypadDecimal   = KeyCode{65, "KeypadDecimal"}
	KeypadMultiply  = KeyCode{67, "KeypadMultiply"}
	KeypadPlus      = KeyCode{69, "KeypadPlus"}
	KeypadClear     = KeyCode{71, "KeypadClear"}
	VolumeUp        = KeyCode{72, "VolumeUp"}
	VolumeDown      = KeyCode{73, "VolumeDown"}
	Mute            = KeyCode{74, "Mute"}
	KeypadDivide    = KeyCode{75, "KeypadDivide"}
	KeypadEnter     = KeyCode{76, "KeypadEnter"}
	KeypadMinus     = KeyCode{78, "KeypadMinus"}
	F18             = KeyCode{79, "F18"}
	F19             = KeyCode{80, "F19"}
	KeypadEquals    = KeyCode{81, "KeypadEquals"}
	Keypad0         = KeyCode{82, "Keypad0"}
	Keypad1         = KeyCode{83, "Keypad1"}
	Keypad2         = KeyCode{84, "Keypad2"}
	Keypad3         = KeyCode{85, "Keypad3"}
	Keypad4         = KeyCode{86, "Keypad4"}
	Keypad5         = KeyCode{87, "Keypad5"}
	Keypad6         = KeyCode{88, "Keypad6"}
	Keypad7         = KeyCode{89, "Keypad7"}
	F20             = KeyCode{90, "F20"}
	Keypad8         = KeyCode{91, "Keypad8"}
	Keypad9         = KeyCode{92, "Keypad9"}
	JIS_Yen         = KeyCode{93, "JIS_Yen"}
	JIS_Underscore  = KeyCode{94, "JIS_Underscore"}
	JIS_KeypadComma = KeyCode{95, "JIS_KeypadComma"}
	F5              = KeyCode{96, "F5"}
	F6              = KeyCode{97, "F6"}
	F7              = KeyCode{98, "F7"}
	F3              = KeyCode{99, "F3"}
	F8              = KeyCode{100, "F8"}
	F9              = KeyCode{101, "F9"}
	JIS_Eisu        = KeyCode{102, "JIS_Eisu"}
	F11             = KeyCode{103, "F11"}
	JIS_Kana        = KeyCode{104, "JIS_Kana"}
	F13             = KeyCode{105, "F13"}
	F16             = KeyCode{106, "F16"}
	F14             = KeyCode{107, "F14"}
	F10             = KeyCode{109, "F10"}
	F12             = KeyCode{111, "F12"}
	F15             = KeyCode{113, "F15"}
	Help            = KeyCode{114, "Help"}
	Home            = KeyCode{115, "Home"}
	PageUp          = KeyCode{116, "PageUp"}
	ForwardDelete   = KeyCode{117, "ForwardDelete"}
	F4              = KeyCode{118, "F4"}
	End             = KeyCode{119, "End"}
	F2              = KeyCode{120, "F2"}
	PageDown        = KeyCode{121, "PageDown"}
	F1              = KeyCode{122, "F1"}
	LeftArrow       = KeyCode{123, "LeftArrow"}
	RightArrow      = KeyCode{124, "RightArrow"}
	DownArrow       = KeyCode{125, "DownArrow"}
	UpArrow         = KeyCode{126, "UpArrow"}
	LeftFunction    = KeyCode{179, "Function"}
)

var Raw2Keycode = map[uint16]KeyCode{
	0:   CHAR_A,
	1:   CHAR_S,
	2:   CHAR_D,
	3:   CHAR_F,
	4:   CHAR_H,
	5:   CHAR_G,
	6:   CHAR_Z,
	7:   CHAR_X,
	8:   CHAR_C,
	9:   CHAR_V,
	10:  ISO_Section,
	11:  CHAR_B,
	12:  CHAR_Q,
	13:  CHAR_W,
	14:  CHAR_E,
	15:  CHAR_R,
	16:  CHAR_Y,
	17:  CHAR_T,
	18:  DIGIT_1,
	19:  DIGIT_2,
	20:  DIGIT_3,
	21:  DIGIT_4,
	22:  DIGIT_6,
	23:  DIGIT_5,
	24:  Equal,
	25:  DIGIT_9,
	26:  DIGIT_7,
	27:  Minus,
	28:  DIGIT_8,
	29:  DIGIT_0,
	30:  RightBracket,
	31:  CHAR_O,
	32:  CHAR_U,
	33:  LeftBracket,
	34:  CHAR_I,
	35:  CHAR_P,
	36:  Return,
	37:  CHAR_L,
	38:  CHAR_J,
	39:  Quote,
	40:  CHAR_K,
	41:  Semicolon,
	42:  Backslash,
	43:  Comma,
	44:  Slash,
	45:  CHAR_N,
	46:  CHAR_M,
	47:  Period,
	48:  Tab,
	49:  Space,
	50:  Grave,
	51:  Delete,
	53:  Escape,
	54:  RightCommand,
	55:  Command,
	56:  Shift,
	57:  CapsLock,
	58:  Option,
	59:  Control,
	60:  RightShift,
	61:  RightOption,
	62:  RightControl,
	63:  Function,
	64:  F17,
	65:  KeypadDecimal,
	67:  KeypadMultiply,
	69:  KeypadPlus,
	71:  KeypadClear,
	72:  VolumeUp,
	73:  VolumeDown,
	74:  Mute,
	75:  KeypadDivide,
	76:  KeypadEnter,
	78:  KeypadMinus,
	79:  F18,
	80:  F19,
	81:  KeypadEquals,
	82:  Keypad0,
	83:  Keypad1,
	84:  Keypad2,
	85:  Keypad3,
	86:  Keypad4,
	87:  Keypad5,
	88:  Keypad6,
	89:  Keypad7,
	90:  F20,
	91:  Keypad8,
	92:  Keypad9,
	93:  JIS_Yen,
	94:  JIS_Underscore,
	95:  JIS_KeypadComma,
	96:  F5,
	97:  F6,
	98:  F7,
	99:  F3,
	100: F8,
	101: F9,
	102: JIS_Eisu,
	103: F11,
	104: JIS_Kana,
	105: F13,
	106: F16,
	107: F14,
	109: F10,
	111: F12,
	113: F15,
	114: Help,
	115: Home,
	116: PageUp,
	117: ForwardDelete,
	118: F4,
	119: End,
	120: F2,
	121: PageDown,
	122: F1,
	123: LeftArrow,
	124: RightArrow,
	125: DownArrow,
	126: UpArrow,
	179: LeftFunction,
}
