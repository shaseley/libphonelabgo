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
		Source:           source.Processor,
		InterlaceZerosMs: int64(interlace),
	}
}

type FrameDiffSample struct {
	SFFrameDiff
	Inserted     bool    `json:"inserted"`
	TraceTimeAdj float64 `json:"trace_time_adj"`
}

func (sample *FrameDiffSample) MonotonicTimestamp() float64 {
	if GlobalConf.UseSysTime {
		return float64(sample.Timestamp) / float64(1000.0)
	} else {
		return sample.TraceTimeAdj
	}
}

// State tracker/unpacker
type FrameDiffEmitter struct {
	Source           phonelab.Processor
	InterlaceZerosMs int64
}

func (emitter *FrameDiffEmitter) Process() <-chan interface{} {

	outChan := make(chan interface{})

	go func() {
		inChan := emitter.Source.Process()

		// Clock skew between different monotonic clocks.
		// Add this to diff timestamps to get trace timestamp.
		curOffsetMs := int64(0)
		prevToken := int64(-1)
		lastTsMs := int64(0)

		var prevDiff *SFFrameDiff

		for iLog := range inChan {
			if ll, ok := iLog.(*phonelab.Logline); ok {
				switch t := ll.Payload.(type) {

				case *SFFrameDiffLog:
					{
						// Check token for things that don't look quite right
						if prevToken >= 0 && prevToken+1 != t.Token {
							fmt.Printf("Warning: Missing tokens. Prev = %v, New = %v\n", prevToken, t.Token)
						}
						prevToken = t.Token

						// Unpack, adjust timestamps, interlace zeros if nec., and send each entry
						for _, diff := range t.Diffs {

							newDiff := &FrameDiffSample{
								SFFrameDiff:  *diff,
								TraceTimeAdj: adjustTimestampMsToS(diff.Timestamp, curOffsetMs),
								Inserted:     false,
							}

							// SurfaceFlinger doesn't swap buffers if no new buffers have been commited,
							// which means we don't always get diffs if the screen hasn't changed.
							// This adds dummy 0.00 diff entries to help downstream algorithms that expect
							// all diffs to be in the stream.

							for emitter.InterlaceZerosMs > 0 && lastTsMs > 0 && prevDiff != nil && newDiff.Timestamp-lastTsMs > 2*emitter.InterlaceZerosMs {
								newTsMs := lastTsMs + emitter.InterlaceZerosMs
								inserted := &FrameDiffSample{
									SFFrameDiff: SFFrameDiff{
										Timestamp: newTsMs,
										PctDiff:   float64(0.0),
										HasColor:  prevDiff.HasColor,
										Mode:      prevDiff.Mode,
									},
									TraceTimeAdj: adjustTimestampMsToS(newTsMs, curOffsetMs),
									Inserted:     true,
								}
								lastTsMs = newTsMs
								outChan <- inserted
							}

							lastTsMs = newDiff.Timestamp

							outChan <- &FrameDiffSample{
								SFFrameDiff:  *diff,
								TraceTimeAdj: adjustTimestampMsToS(diff.Timestamp, curOffsetMs),
								Inserted:     false,
							}

							prevDiff = diff
						}
					}
				case *TimeSyncMsg:
					{
						// Update current time offset
						curOffsetMs = t.OffsetNs / nsPerMs
					}
				}
			}
		}
		close(outChan)
	}()

	return outChan
}
