package libphonelabgo

import (
	"fmt"
	phonelab "github.com/shaseley/phonelab-go"
)

type FrameRefreshEvent struct {
	SysTimeNs    int64
	TraceTimeAdj float64
}

func (event *FrameRefreshEvent) MonotonicTimestamp() float64 {
	if GlobalConf.UseSysTime {
		return float64(event.SysTimeNs) / nsPerSecF
	} else {
		return event.TraceTimeAdj
	}
}

type FrameRefreshEmitter struct {
	Source phonelab.Processor
}

func (emitter *FrameRefreshEmitter) Process() <-chan interface{} {

	outChan := make(chan interface{})

	go func() {
		inChan := emitter.Source.Process()

		// Clock skew between different monotonic clocks.
		// Add this to diff timestamps to get trace timestamp.
		curOffsetNs := int64(0)
		prevToken := int64(-1)

		for iLog := range inChan {
			if ll, ok := iLog.(*phonelab.Logline); ok {
				switch t := ll.Payload.(type) {

				case *SFFrameTimesLog:
					{
						// Check token for things that don't look quite right
						if prevToken >= 0 && prevToken+1 != t.Token {
							fmt.Printf("Warning: Missing tokens. Prev = %v, New = %v\n", prevToken, t.Token)
						}
						prevToken = t.Token

						// Unpack
						for _, sysTs := range t.Times {
							outChan <- &FrameRefreshEvent{
								SysTimeNs:    sysTs,
								TraceTimeAdj: float64(sysTs+curOffsetNs) / nsPerSecF,
							}
						}
					}

				case *TimeSyncMsg:
					{
						// Update current time offset
						curOffsetNs = t.OffsetNs
					}
				}
			}
		}
	}()

	return outChan
}

type FrameRefreshEmiiterGen struct{}

func (g *FrameRefreshEmiiterGen) GenerateProcessor(source *phonelab.PipelineSourceInstance,
	kwargs map[string]interface{}) phonelab.Processor {

	return &FrameRefreshEmitter{
		Source: source.Processor,
	}
}
