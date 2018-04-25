package libphonelabgo

import (
	phonelab "github.com/shaseley/phonelab-go"
)

// There can be clock skew between the jiffy-based tracetime monotonic clock and
// systemTime() which uses the POSIX monotonic clock. This processor emits
// offests to enable tighter time sync.
type TimeSyncPreprocessor struct {
	Source phonelab.Processor
}

const (
	msPerSec  = int64(1 * 1000)
	usPerSec  = int64(1 * 1000 * 1000)
	nsPerSec  = int64(1 * 1000 * 1000 * 1000)
	nsPerMs   = int64(1 * 1000 * 1000)
	msPerSecF = float64(msPerSec)
	usPerSecF = float64(usPerSec)
	nsPerSecF = float64(nsPerSec)
	nsPerMsF  = float64(nsPerMs)
)

type TimeSyncMsg struct {
	OffsetNs    int64
	TraceTimeNs int64
	SysTimeNs   int64
}

func adjustTimestamp(ts, offset, unitsPerSec int64) float64 {
	ts += offset
	secs := ts / unitsPerSec
	rem := ts - (secs * unitsPerSec)
	return float64(secs) + (float64(rem) / float64(unitsPerSec))
}

func adjustTimestampMsToS(ts, offset int64) float64 {
	return adjustTimestamp(ts, offset, msPerSec)
}

func (p *TimeSyncPreprocessor) Process() <-chan interface{} {

	outChan := make(chan interface{})

	go func() {
		inChan := p.Source.Process()

		// Clock skew between different monotonic clocks.
		// Add this to diff timestamps to get trace timestamp.
		curOffset := int64(0)

		for iLog := range inChan {
			if ll, ok := iLog.(*phonelab.Logline); ok && ll != nil {
				if fpsLog, ok := ll.Payload.(*SFFpsLog); ok {
					// Update current time offset
					traceTsNanos := int64(ll.TraceTime * nsPerSecF)
					if newOffset := traceTsNanos - fpsLog.SysTimestamp; newOffset != curOffset {
						traceTsNanos := int64(ll.TraceTime * nsPerSecF)
						curOffset = newOffset
						outChan <- &TimeSyncMsg{
							OffsetNs:    curOffset,
							TraceTimeNs: traceTsNanos,
							SysTimeNs:   fpsLog.SysTimestamp,
						}
					}
				}
				outChan <- iLog
			}
		}
		close(outChan)
	}()

	return outChan
}

type TimeSyncPreprocessorGenerator struct{}

func (g *TimeSyncPreprocessorGenerator) GenerateProcessor(source *phonelab.PipelineSourceInstance,
	kwargs map[string]interface{}) phonelab.Processor {

	return &TimeSyncPreprocessor{
		Source: source.Processor,
	}

}
