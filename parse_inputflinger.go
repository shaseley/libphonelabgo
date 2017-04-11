package libphonelabgo

import (
	phonelab "github.com/shaseley/phonelab-go"
)

type IFKeyEventLog struct {
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

type IFMotionEventLog struct {
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
