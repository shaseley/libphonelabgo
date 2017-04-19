package libphonelabgo

import (
	phonelab "github.com/shaseley/phonelab-go"
)

// InputServiceManager (IMS) lifecycle logs
type IMSLifeCycleLog struct {
	phonelab.PLLog
	Action   string `json:"Action"`
	UpTimeNs int64  `json:"UpTimeNs"`
}

type IMSLifeCycleLogProps struct{}

func (p *IMSLifeCycleLogProps) New() interface{} {
	return &IMSLifeCycleLog{}
}

func NewIMSLifeCycleParser() phonelab.Parser {
	return phonelab.NewJSONParser(&IMSLifeCycleLogProps{})
}
