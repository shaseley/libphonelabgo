package libphonelabgo

import (
	"fmt"
	phonelab "github.com/shaseley/phonelab-go"
)

// InputStateMachine parameters. These control what we consider local vs.
// global response, jankiness, timeouts, etc.
type InputStateMachineParams struct {
	JankThresholdMs       int64
	LocalResponsePercent  float64
	GlobalResponsePercent float64
	GlobalResponseRegions int
	UITimeoutMs           int64
	Connectivity          PixelConnectivity
}

// Create a new InputStateMachineParams with the default settings.
func DefaultInputStateMachineParams() *InputStateMachineParams {
	return &InputStateMachineParams{
		JankThresholdMs:       70,
		LocalResponsePercent:  75.0,
		GlobalResponsePercent: 20.0,
		GlobalResponseRegions: 10,
		UITimeoutMs:           5000,
		Connectivity:          FourConnected,
	}
}

// InputStateMachine states
const (
	InputStateWaitInput = iota
	InputStateWaitResponse
	InputStateMeasureLocal
	InputStateMeasureGlobal
)

// InputStateMachine is the state machine we use to measure performance metrics
// of an input event. For now, we only support taps, but this can (and should)
// be extended to handle scrolls as well. Eventually, we'd like to have a state
// machine that models all basic interactions.
type InputStateMachine struct {
	// Parameters
	Params *InputStateMachineParams

	// State
	curState  int
	curResult *TapEventResult
	curEvent  *TouchScreenEvent
}

// Create a new InputStateMachine with the default parameters.
func NewInputStateMachine() *InputStateMachine {
	return &InputStateMachine{
		Params: DefaultInputStateMachineParams(),
	}
}

// JankEvent defines one instance in a measured response where consecutive
// inter-frame time delta is above some threshold. In practice, there are
// different types of jank, but they are all measured in the same way.
type JankEvent struct {
	TimestampNs int64 `json:"timestamp_ns"`
	JankAmount  int64 `json:"jank_amount"`
}

const (
	InvalidResponseTime = -1
)

const (
	TapEventFinishTimeout = iota
	TapEventFinishShortCircuit
)

// TapEventResult encapsulates the response detail and performance metrics of a
// single tap event.
type TapEventResult struct {
	TimestampNs    int64           `json:"timestamp_ns"`
	FinishNs       int64           `json:"finish_ns"`
	FinishType     int             `json:"finish_type"`
	LocalResponse  *ResponseDetail `json:"local_response"`
	GlobalResponse *ResponseDetail `json:"global_response"`
}

func NewTapEventResult(event *TouchScreenEvent) *TapEventResult {
	return &TapEventResult{
		TimestampNs:    event.Timestamp,
		LocalResponse:  NewResponseDetail(),
		GlobalResponse: NewResponseDetail(),
	}
}

// ResponseDetail contains the information needed to capture the performance
// metrics for either a local or global response to an input event.
type ResponseDetail struct {
	StartNs int64        `json:"start_ns"`
	EndNs   int64        `json:"end_ns"`
	Jank    []*JankEvent `json:"jank_events"`

	params        *InputStateMachineParams `json:"-"`
	prevFrameTime int64
}

func NewResponseDetail() *ResponseDetail {
	return &ResponseDetail{
		Jank:    make([]*JankEvent, 0),
		StartNs: InvalidResponseTime,
		EndNs:   InvalidResponseTime,
	}
}

// Update the response with a new diff sample.
func (response *ResponseDetail) onFrameDiff(diff *FrameDiffSample) {
	if diff.PctDiff == 0.0 {
		// Don't advance the state at all.
		return
	}

	// Advance the end time for this response
	response.EndNs = diff.Timestamp
}

// Update the response with a new screen refresh event and check for jank.
func (response *ResponseDetail) onFrameRefresh(event *FrameRefreshEvent, jankThresholdMs int64) {
	if response.prevFrameTime > 0 {
		diff := (event.SysTimeNs - response.prevFrameTime) / 1000000
		if diff >= jankThresholdMs {
			response.Jank = append(response.Jank, &JankEvent{
				TimestampNs: event.SysTimeNs,
				JankAmount:  diff,
			})
		}
	}
	response.prevFrameTime = event.SysTimeNs
}

// Short-circuit the current result/analysis. This gets called when we're
// analyzing the post-tap stream and another input event comes along.
func (ism *InputStateMachine) shortCircuit(ts int64) *TapEventResult {
	if ism.curResult == nil {
		return nil
	}

	// Finish detail
	ism.curResult.FinishType = TapEventFinishShortCircuit
	ism.curResult.FinishNs = ts

	// TODO: Do we need to detect timeouts here?
	// If the diffs interlace zeros, then probably not since we'll have
	// frequent timestamps.

	return ism.curResult
}

// State change InputStateWaitResponse --> InputStateWaitInput
func (ism *InputStateMachine) handleTimeout(ts int64) *TapEventResult {
	res := ism.curResult
	res.FinishType = TapEventFinishTimeout
	res.FinishNs = ts

	// --> InputStateWaitInput
	ism.reset()

	return res
}

// Reset to the start state (--> InputStateWaitInput).
func (ism *InputStateMachine) reset() {
	ism.curState = InputStateWaitInput
	ism.curEvent = nil
	ism.curResult = nil
}

type responseType int

const (
	responseTypeLocal responseType = iota
	responseTypeGlobal
	responseTypeNone
	responseTypeNeither
)

// Determine the responseType of the frame diff
func (ism *InputStateMachine) getResponseType(diff *FrameDiffSample) responseType {
	if diff.PctDiff == 0.0 {
		return responseTypeNone
	}

	localPctDiff, localPctNormalized, err := diff.LocalDiff(ism.Params.Connectivity, ism.curEvent.X, ism.curEvent.Y)
	if err != nil {
		// Not good.
		panic(err)
	}

	// FIXME: Is this approach reasonable?

	// If the local diff is non-zero and a substantial percentage of the
	// total diff, we consider it a local response.
	if localPctDiff > 0 {
		ratio := 100.0 * (localPctNormalized / diff.PctDiff)
		if ratio >= ism.Params.LocalResponsePercent {
			return responseTypeLocal
		}
	}

	// Not a local diff
	if diff.PctDiff >= ism.Params.GlobalResponsePercent || len(diff.GridEntries) >= ism.Params.GlobalResponseRegions {
		return responseTypeGlobal
	}

	return responseTypeNeither
}

// Update state and possibly return an event result
func (ism *InputStateMachine) OnTouchEvent(event *TouchScreenEvent) *TapEventResult {

	// Skip key events
	if event.What == TouchScreenEventKey {
		return nil
	}

	// For all other touch events, this short-circuits the current state if
	// we're not at the start/wait state.
	var cur *TapEventResult = nil

	if ism.curState != InputStateWaitInput {
		cur = ism.shortCircuit(event.Timestamp)
	}

	// Clear state
	ism.reset()

	// If it's a tap event, transition --> InputStateWaitResponse
	if event.What == TouchScreenEventTap {
		ism.startWaitingForResponse(event)
	}

	return cur
}

// State change from InputStateWaitInput --> InputStateWaitResponse
func (ism *InputStateMachine) startWaitingForResponse(event *TouchScreenEvent) {
	ism.curState = InputStateWaitResponse
	ism.curEvent = event
	ism.curResult = NewTapEventResult(event)
}

// State change from InputStateWaitResponse --> InputStateMeasureLocal
func (ism *InputStateMachine) startMeasuringLocalRepsonse(diff *FrameDiffSample) {
	ism.curState = InputStateMeasureLocal
	ism.curResult.LocalResponse.StartNs = diff.Timestamp
	ism.curResult.LocalResponse.EndNs = diff.Timestamp
}

// State change from InputStateWaitResponse or InputStateMeasureLocal --> InputStateMeasureGlobal
func (ism *InputStateMachine) startMeasuringGlobalRepsonse(diff *FrameDiffSample) {
	ism.curState = InputStateMeasureGlobal
	ism.curResult.GlobalResponse.StartNs = diff.Timestamp
	ism.curResult.GlobalResponse.EndNs = diff.Timestamp
}

// Update state and possibly return an event result.
func (ism *InputStateMachine) OnFrameDiff(diff *FrameDiffSample) *TapEventResult {

	// Short circuit: we don't do anything with diffs if we're waiting for
	// input.
	if ism.curState == InputStateWaitInput {
		return nil
	}

	// We have a response (which could be 0.0%), but what type is it?
	rt := ism.getResponseType(diff)

	// Handle timeouts in one shot
	if rt == responseTypeNone {
		ns := int64(0)
		switch ism.curState {
		case InputStateWaitResponse:
			ns = ism.curResult.TimestampNs
		case InputStateMeasureLocal:
			ns = ism.curResult.LocalResponse.EndNs
		case InputStateMeasureGlobal:
			ns = ism.curResult.GlobalResponse.EndNs
		}

		if ns > 0 && (diff.Timestamp-ns)/1000000 >= ism.Params.UITimeoutMs {
			// We timed out
			return ism.handleTimeout(diff.Timestamp)
		} else {
			// Otherwise, no change
			return nil
		}
	}

	// OK, we have a resonse of some sort and we're not in the start state.
	// We'll either update the current response, or transition to a different
	// starte.

	switch ism.curState {
	default:
		{
			panic(fmt.Sprint("Unexpected state: %v", ism.curState))
		}
	case InputStateWaitResponse:
		{
			if rt == responseTypeLocal {
				// State transition to wait start measuring local response
				ism.startMeasuringLocalRepsonse(diff)
			} else {
				// TODO: This is a simplified -- we're considering any
				// non-local response as a global response. In practice, we
				// could have an animated transition which we shouldn't count.
				// For now, the spinner detection module handles that and we
				// evaluate it separately.

				// State transition to wait start measuring global response
				ism.startMeasuringGlobalRepsonse(diff)
			}
		}
	case InputStateMeasureLocal:
		{
			if rt == responseTypeLocal {
				// Keep measuring local
				ism.curResult.LocalResponse.onFrameDiff(diff)
			} else {
				// State transition to start measuring the global response
				ism.startMeasuringGlobalRepsonse(diff)
			}
		}
	case InputStateMeasureGlobal:
		{
			// Even if the diff is only local, we had a global response and we
			// don't flip-flop states.
			ism.curResult.GlobalResponse.onFrameDiff(diff)
		}
	}

	return nil
}

func (ism *InputStateMachine) OnFrameRefresh(event *FrameRefreshEvent) *TapEventResult {
	// At this point, we'll only use this info to update the jankiness, so we'll always
	// return nil.

	switch ism.curState {
	default:
		{
			return nil
		}
	case InputStateMeasureLocal:
		{
			ism.curResult.LocalResponse.onFrameRefresh(event, ism.Params.JankThresholdMs)
		}

	case InputStateMeasureGlobal:
		{
			ism.curResult.GlobalResponse.onFrameRefresh(event, ism.Params.JankThresholdMs)
		}
	}
	return nil
}

// Called when the input log stream is finished
func (ism *InputStateMachine) Finish(ts int64) *TapEventResult {
	if ism.curState != InputStateWaitInput {
		// Just short-circuit the current test in the same way as if we would
		// received a new touch event.
		return ism.shortCircuit(ts)
	}
	return nil
}

type InputStateMachineProcessor struct {
	Source phonelab.Processor
}

func (proc *InputStateMachineProcessor) Process() <-chan interface{} {

	outChan := make(chan interface{})
	inChan := proc.Source.Process()

	go func() {
		ism := NewInputStateMachine()
		var lastTs int64

		for iLog := range inChan {
			// We're only expecting frame diffs and input logs
			switch t := iLog.(type) {
			case *TouchScreenEvent:
				{
					if res := ism.OnTouchEvent(t); res != nil {
						outChan <- res
					}
				}

			case *FrameDiffSample:
				{
					// TODO: This should be done elsewhere
					(&t.SFFrameDiff).initScreenGrid(allScreenGrids[0])
					if res := ism.OnFrameDiff(t); res != nil {
						outChan <- res
					}
				}
			case *FrameRefreshEvent:
				{
					lastTs = t.SysTimeNs

					if res := ism.OnFrameRefresh(t); res != nil {
						outChan <- res
					}
				}
			}
		}

		if res := ism.Finish(lastTs); res != nil {
			outChan <- res
		}

		// Done.
		close(outChan)
	}()

	return outChan
}
