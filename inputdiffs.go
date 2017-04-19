package libphonelabgo

import (
	"fmt"
	phonelab "github.com/shaseley/phonelab-go"
)

// InputDiffProcessor is a Processor that combines input events with nearby
// diff samples.
type InputDiffProcessor struct {
	DiffDurationMs int64
	Source         phonelab.Processor
}

type InputDiffSample struct {
	Timestamp  int64
	LocalDiff  float64
	GlobalDiff float64
}

type InputDiffEvent struct {
	EventDetail *TouchScreenEvent
	Eclipsed    bool
	Diffs       []*InputDiffSample
}

func (proc *InputDiffProcessor) Process() <-chan interface{} {

	outChan := make(chan interface{})
	inChan := proc.Source.Process()

	var curEvent *InputDiffEvent

	go func() {
		for iLog := range inChan {
			// We're only expecting frame diffs and input logs
			switch t := iLog.(type) {
			case *TouchScreenEvent:
				{
					// New event
					if curEvent != nil {
						curEvent.Eclipsed = true
						outChan <- curEvent
					}
					if t.What == TouchScreenEventTap {
						curEvent = &InputDiffEvent{
							EventDetail: t,
							Diffs:       make([]*InputDiffSample, 0),
						}
					} else {
						curEvent = nil
					}
				}

			case *FrameDiffSample:
				{
					if curEvent == nil {
						continue
					}

					// TODO: This should be done in the diffstream
					(&t.SFFrameDiff).initScreenGrid(allScreenGrids[0])

					diffTsMs := t.SFFrameDiff.Timestamp
					curEventMs := curEvent.EventDetail.Timestamp / 1000000

					if diffTsMs-curEventMs <= proc.DiffDurationMs {

						localDiff, err := t.SFFrameDiff.LocalDiff(FourConnected,
							curEvent.EventDetail.X, curEvent.EventDetail.Y)

						if err != nil {
							// TODO: Wrong logger
							fmt.Println("Error getting local diff:", err)
						} else {
							curEvent.Diffs = append(curEvent.Diffs, &InputDiffSample{
								Timestamp:  t.SFFrameDiff.Timestamp,
								LocalDiff:  localDiff,
								GlobalDiff: t.SFFrameDiff.PctDiff,
							})
						}
					} else {
						outChan <- curEvent
						curEvent = nil
					}

				}
			}
		}

		// Send the final input event, if there is one
		if curEvent != nil {
			outChan <- curEvent
		}
		// Done.
		close(outChan)
	}()

	return outChan
}

////////////////////////////////////////////////////////////////////////////////

const DefaultDiffDuration = 5000

type InputDiffProcessorGenerator struct{}

func (g *InputDiffProcessorGenerator) GenerateProcessor(source *phonelab.PipelineSourceInstance,
	kwargs map[string]interface{}) phonelab.Processor {

	diffDuration := DefaultDiffDuration

	if val, ok := kwargs["diff_duration_ms"]; ok {
		if diffDuration, ok = val.(int); !ok {
			fmt.Printf("Warning: wrong type for 'diff_duration_ms' (*T)\n", diffDuration)
		}
	}

	return &InputDiffProcessor{
		Source:         source.Processor,
		DiffDurationMs: int64(diffDuration),
	}
}
