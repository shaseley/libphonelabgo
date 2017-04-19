package libphonelabgo

import (
	phonelab "github.com/shaseley/phonelab-go"
)

const (
	IMSPhoneLabTag = "InputMethodService-LifeCycle-QoE"
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

// Register the ISM parser with the environment
func AddIMSParser(env *phonelab.Environment) {
	env.RegisterParserGenerator(IMSPhoneLabTag, NewIMSLifeCycleParser)
}
