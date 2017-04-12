package libphonelabgo

import (
	phonelab "github.com/shaseley/phonelab-go"
)

type InputProcessor struct {
	TouchSlop int
	Source    phonelab.Processor
}

func (p *InputProcessor) Process() <-chan interface{} {
	outChan := make(chan interface{})
	inChan := p.Source.Process()

	go func() {

		detector := NewGestureDetector(p.TouchSlop)

		for raw := range inChan {
			if log, ok := raw.(*phonelab.Logline); ok && log != nil {
				switch typed := log.Payload.(type) {
				case *IFKeyEventLog:
					{
						// Just emit the key event if it is an up
						if typed.Action == KEY_ACTION_UP {
							event := &TouchScreenEvent{
								What:      TouchScreenEventKey,
								Timestamp: typed.Timestamp,
								TraceTime: log.TraceTime,
								Code:      typed.KeyCode,
							}
							outChan <- event
						}
					}
				case *IFMotionEventLog:
					{
						// Update the detector state
						if outEvent, err := detector.OnTouchEvent(log.TraceTime, typed); err != nil {
							panic(err)
						} else if outEvent != nil {
							outChan <- outEvent
						}
					}
				}
			}
		}
		close(outChan)
	}()

	return outChan
}

type InputProcessorGenerator struct{}

func (ipg *InputProcessorGenerator) GenerateProcessor(source *phonelab.PipelineSourceInstance,
	kwargs map[string]interface{}) phonelab.Processor {

	return &InputProcessor{
		TouchSlop: TouchSlopScaled,
		Source:    source.Processor,
	}
}
