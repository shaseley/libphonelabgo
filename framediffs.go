package libphonelabgo

import (
	"fmt"
	phonelab "github.com/shaseley/phonelab-go"
)

// TODO: should rely on charge state

type FrameDiffEmitterGenerator struct{}

func (g *FrameDiffEmitterGenerator) GenerateProcessor(source *phonelab.PipelineSourceInstance,
	kwargs map[string]interface{}) phonelab.Processor {

	interlace := 0
	if val, ok := kwargs["interlace"]; ok {
		if interlace, ok = val.(int); !ok {
			fmt.Printf("Warning: wrong type for 'interlace' (*T)\n", interlace)
		}
	}

	return &FrameDiffEmitter{
		Source:         source.Processor,
		InterlaceZeros: int64(interlace),
	}
}

type FrameDiffSample struct {
	SFFrameDiff
	Inserted     bool    `json:"inserted"`
	TraceTimeAdj float64 `json:"trace_time_adj"`
}

func (sample *FrameDiffSample) MonotonicTimestamp() float64 {
	return sample.TraceTimeAdj
}

// State tracker/unpacker
type FrameDiffEmitter struct {
	Source         phonelab.Processor
	InterlaceZeros int64
}

const nsPerSec = int64(1000000000)
const nsPerSecF = float64(1000000000.0)

func adjustTimestampNsToS(ts int64, offset int64) float64 {
	ts += offset
	secs := ts / nsPerSec
	ns := ts - (secs * nsPerSec)
	return float64(secs) + (float64(ns) / nsPerSecF)
}

func (emitter *FrameDiffEmitter) Process() <-chan interface{} {

	outChan := make(chan interface{})

	go func() {
		inChan := emitter.Source.Process()

		// Clock skew between different monotonic clocks.
		// Add this to diff timestamps to get trace timestamp.
		curOffset := int64(0)
		prevToken := int64(-1)

		// TODO: Inerlace zeros
		//lastTimestamp := float64(0)

		for iLog := range inChan {
			if ll, ok := iLog.(*phonelab.Logline); ok {
				switch t := ll.Payload.(type) {
				case *SFFrameDiffLog:
					// Check token for things that don't look quite right
					if prevToken >= 0 && prevToken+1 != t.Token {
						fmt.Printf("Warning: Missing tokens. Prev = %v, New = %v\n", prevToken, t.Token)
					}

					prevToken = t.Token

					// Unpack, adjust timestamps, send each entry
					for _, diff := range t.Diffs {

						newDiff := &FrameDiffSample{
							SFFrameDiff:  *diff,
							TraceTimeAdj: adjustTimestampNsToS(diff.Timestamp, curOffset),
							Inserted:     false,
						}

						// TODO: Inerlace zeros
						//lastTimestamp = newDiff.TraceTimeAdj

						outChan <- newDiff
					}

					break
				case *SFFpsLog:
					{
						// Update current time offset
						traceTsNanos := int64(ll.TraceTime * nsPerSecF)
						curOffset = traceTsNanos - t.SysTimestamp
						break
					}
				}
			}
		}
		close(outChan)
	}()

	return outChan
}
