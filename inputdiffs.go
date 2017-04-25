package libphonelabgo

import (
	"fmt"
	phonelab "github.com/shaseley/phonelab-go"
)

// InputDiffProcessor is a Processor that combines input events with nearby
// diff samples.
type InputDiffProcessor struct {
	Args   *InputDiffProcessorArgs
	Source phonelab.Processor
}

type LocalDiffSample struct {
	Diff       float64 `json:"diff"`
	GlobalDiff float64 `json:"global_diff"`
}

type InputDiffSample struct {
	Timestamp  int64            `json:"timestamp"`
	LocalDiff1 *LocalDiffSample `json:"local_diff1"`
	LocalDiff4 *LocalDiffSample `json:"local_diff4"`
	LocalDiff8 *LocalDiffSample `json:"local_diff8"`
	GlobalDiff float64          `json:"global_diff"`
	NumChanges int              `json:"num_changes"`
}

type InputDiffEvent struct {
	EventDetail []*TouchScreenEvent `json:"event_detail"`
	Eclipsed    bool                `json:"eclipsed"`
	Diffs       []*InputDiffSample  `json:"diffs"`
	complete    bool
}

const DefaultDiffDuration = 5000

type InputDiffProcessorArgs struct {
	DoTaps         bool
	DoKeys         bool
	DoScrolls      bool
	DiffDurationMs int64
}

func NewInputDiffProcessorArgs(kwargs map[string]interface{}) *InputDiffProcessorArgs {
	args := &InputDiffProcessorArgs{
		DiffDurationMs: DefaultDiffDuration,
	}

	if v, ok := kwargs["do_taps"]; ok {
		args.DoTaps, _ = v.(bool)
	}

	if v, ok := kwargs["do_scrolls"]; ok {
		args.DoScrolls, _ = v.(bool)
	}

	if v, ok := kwargs["do_keys"]; ok {
		args.DoKeys, _ = v.(bool)
	}

	if v, ok := kwargs["diff_duration_ms"]; ok {
		var diffDuration int
		if diffDuration, ok = v.(int); !ok {
			fmt.Printf("Warning: wrong type for 'diff_duration_ms' (*T)\n", diffDuration)
		}
		args.DiffDurationMs = int64(diffDuration)
	}

	return args
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
					if curEvent != nil {
						if curEvent.complete {
							curEvent.Eclipsed = true
							outChan <- curEvent
							curEvent = nil
						} else if curEvent.EventDetail[0].What == TouchScreenEventScrollStart {
							switch t.What {
							default:
								// Bad state, just discard
								curEvent = nil
							case TouchScreenEventScroll:
								// Keep going with this event
								curEvent.EventDetail = append(curEvent.EventDetail, t)
							case TouchScreenEventScrollEnd:
								curEvent.EventDetail = append(curEvent.EventDetail, t)
								curEvent.complete = true
							}
						} else {
							panic("Unexpected condition in InputDiffProcessor")
						}
					}

					if curEvent == nil {
						if (t.What == TouchScreenEventTap && proc.Args.DoTaps) ||
							(t.What == TouchScreenEventKey && proc.Args.DoKeys) ||
							(t.What == TouchScreenEventScrollStart && proc.Args.DoScrolls) {

							curEvent = &InputDiffEvent{
								EventDetail: []*TouchScreenEvent{t},
								Diffs:       make([]*InputDiffSample, 0),
								complete:    t.What != TouchScreenEventScrollStart,
							}
						}
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
					curDetail := curEvent.EventDetail[len(curEvent.EventDetail)-1]
					curEventMs := curDetail.Timestamp / 1000000

					if diffTsMs-curEventMs <= proc.Args.DiffDurationMs || !curEvent.complete {

						allConn := []PixelConnectivity{OneConnected, FourConnected, EightConnected}
						diffs := make([]*LocalDiffSample, 3, 3)

						for i, conn := range allConn {
							localDiff, sz, err := t.SFFrameDiff.LocalDiff(conn,
								curDetail.X, curDetail.Y)
							if err != nil {
								// TODO: Log
								panic(fmt.Sprintf("Error getting local diff: %v", err))
							}
							diffs[i] = &LocalDiffSample{localDiff, sz}
						}

						curEvent.Diffs = append(curEvent.Diffs, &InputDiffSample{
							Timestamp:  t.SFFrameDiff.Timestamp,
							LocalDiff1: diffs[0],
							LocalDiff4: diffs[1],
							LocalDiff8: diffs[2],
							GlobalDiff: t.SFFrameDiff.PctDiff,
							NumChanges: len(t.SFFrameDiff.GridEntries),
						})

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

type InputDiffProcessorGenerator struct{}

func (g *InputDiffProcessorGenerator) GenerateProcessor(source *phonelab.PipelineSourceInstance,
	kwargs map[string]interface{}) phonelab.Processor {

	return &InputDiffProcessor{
		Source: source.Processor,
		Args:   NewInputDiffProcessorArgs(kwargs),
	}
}
