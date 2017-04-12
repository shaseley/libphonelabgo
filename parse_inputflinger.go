package libphonelabgo

import (
	phonelab "github.com/shaseley/phonelab-go"
)

type IFKeyEventLog struct {
	Seq         int64  `json:"seq"`
	Msg         string `json:"msg"`
	Timestamp   int64  `json:"ts"`
	DeviceId    int    `json:"dev"`
	Source      int    `json:"src"`
	PolicyFlags int    `json:"pflags"`
	Action      int    `json:"action"`
	Flags       int    `json:"flags"`
	KeyCode     int    `json:"key_code"`
	ScanCode    int    `json:"scan_code"`
	MetaState   int    `json:"meta"`
	RepeatCount int    `json:"repeat"`
	Downtime    int64  `json:"dtime"`
}

const (
	KEY_ACTION_DOWN = 0
	KEY_ACTION_UP   = 1
)

type IFPointerData struct {
	Id          int     `json:"id"`
	ToolType    int     `json:"tool"`
	XPos        float64 `json:"x"`
	YPos        float64 `json:"y"`
	Pressure    float64 `json:"pr"`
	Size        float64 `json:"sz"`
	TouchMajor  float64 `json:"tch_mj"`
	TouchMinor  float64 `json:"tch_mn"`
	ToolMajor   float64 `json:"tl_mj"`
	ToolMinor   float64 `json:"tl_mn"`
	Orientation float64 `json:"orient"`
}

// The following constants are taken directly from the Android java code in
// frameworks/base/core/java/android/view/MotionEvent.java.

const ACTION_MASK = 0xff

const (
	ACTION_DOWN                = 0
	ACTION_UP                  = 1
	ACTION_MOVE                = 2
	ACTION_CANCEL              = 3
	ACTION_OUTSIDE             = 4
	ACTION_POINTER_DOWN        = 5
	ACTION_POINTER_UP          = 6
	ACTION_HOVER_MOVE          = 7
	ACTION_SCROLL              = 8
	ACTION_HOVER_ENTER         = 9
	ACTION_HOVER_EXIT          = 10
	ACTION_BUTTON_PRESS        = 11
	ACTION_BUTTON_RELEASE      = 12
	ACTION_POINTER_INDEX_MASK  = 0xff00
	ACTION_POINTER_INDEX_SHIFT = 8
	ACTION_POINTER_1_DOWN      = ACTION_POINTER_DOWN | 0x0000
	ACTION_POINTER_2_DOWN      = ACTION_POINTER_DOWN | 0x0100
	ACTION_POINTER_3_DOWN      = ACTION_POINTER_DOWN | 0x0200
	ACTION_POINTER_1_UP        = ACTION_POINTER_UP | 0x0000
	ACTION_POINTER_2_UP        = ACTION_POINTER_UP | 0x0100
	ACTION_POINTER_3_UP        = ACTION_POINTER_UP | 0x0200
	ACTION_POINTER_ID_MASK     = 0xff00
	ACTION_POINTER_ID_SHIFT    = 8
)

const (
	FLAG_WINDOW_IS_OBSCURED           = 0x1
	FLAG_WINDOW_IS_PARTIALLY_OBSCURED = 0x2
	FLAG_TAINTED                      = 0x80000000
	FLAG_TARGET_ACCESSIBILITY_FOCUS   = 0x40000000
)

const (
	EDGE_TOP    = 0x00000001
	EDGE_BOTTOM = 0x00000002
	EDGE_LEFT   = 0x00000004
	EDGE_RIGHT  = 0x00000008
)

// The following constants are taken directly from the Android java code in
// frameworks/base/core/java/android/view/KeyEvent.java. AFAIK, these are the
// only hard keycodes we'll see in the InputDispatcher on the Nexus 6.
const (
	KEYCODE_HOME        = 3
	KEYCODE_BACK        = 4
	KEYCODE_VOLUME_UP   = 24
	KEYCODE_VOLUME_DOWN = 25
	KEYCODE_POWER       = 26
	KEYCODE_SEARCH      = 84
)

const (
	// This gets set in frameworks/base/core/java/android/view/ViewConfiguration.java
	// based on device config. For the Nexus 6, this works out to 28 pixels. This is
	// esentially the wiggle room we have in any direction for a click.
	TouchSlopScaled = 28
)

type IFMotionEventLog struct {
	Seq          int64            `json:"seq"`
	Msg          string           `json:"msg"`
	Timestamp    int64            `json:"ts"`
	DeviceId     int              `json:"dev"`
	Source       int              `json:"src"`
	PolicyFlags  int              `json:"pflags"`
	Action       int              `json:"action"`
	ActionButton int              `json:"a_btn"`
	Flags        int              `json:"flags"`
	MetaState    int              `json:"meta"`
	ButtonState  int              `json:"btn_state"`
	EdgeFlags    int              `json:"eflags"`
	XPrecision   float64          `json:"x_p"`
	YPrecision   float64          `json:"y_p"`
	Downtime     int              `json:"dtime"`
	PointerData  []*IFPointerData `json:"ptrs"`
}

func (event *IFMotionEventLog) GetActionIndex() int {
	return (event.Action & ACTION_POINTER_INDEX_MASK) >> ACTION_POINTER_INDEX_SHIFT
}

func (event *IFMotionEventLog) GetMaskedAction() int {
	return event.Action & ACTION_MASK
}

type IFMotionEventParserProps struct{}

func (p *IFMotionEventParserProps) New() interface{} {
	return &IFMotionEventLog{}
}

func NewIFMotionEventParser() phonelab.Parser {
	return phonelab.NewJSONParser(&IFMotionEventParserProps{})
}

type IFKeyEventParserProps struct{}

func (p *IFKeyEventParserProps) New() interface{} {
	return &IFKeyEventLog{}
}

func NewIFKeyEventParser() phonelab.Parser {
	return phonelab.NewJSONParser(&IFKeyEventParserProps{})
}

func RegisterInputFlingerParsers(env *phonelab.Environment) {
	env.Parsers["InputDispatcher-MotionEvent"] = func() phonelab.Parser {
		return NewIFMotionEventParser()
	}
	env.Parsers["InputDispatcher-KeyEvent"] = func() phonelab.Parser {
		return NewIFKeyEventParser()
	}
}
