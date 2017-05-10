package libphonelabgo

import (
	"fmt"
	phonelab "github.com/shaseley/phonelab-go"
)

var ismDebug = false
var ignoreFrameTimes = true

// InputStateMachine parameters. These control what we consider local vs.
// global response, jankiness, timeouts, etc.
type InputStateMachineParams struct {
	JankThresholdMs       int64
	JankFilterValue       float64
	LocalResponsePercent  float64
	LocalResponseRegions  int
	GlobalResponsePercent float64
	GlobalResponseRegions int
	UITimeoutMs           int64
	Connectivity          PixelConnectivity
	UsePendingTimestamp   bool
	SkipUndefinedResponse bool
	JankOnFrameUpdate     bool
}

// Create a new InputStateMachineParams with the default settings.
func DefaultInputStateMachineParams() *InputStateMachineParams {
	return &InputStateMachineParams{
		JankThresholdMs:       70,
		JankFilterValue:       0.1,
		LocalResponsePercent:  60.0,
		LocalResponseRegions:  0,
		GlobalResponsePercent: 20.0,
		GlobalResponseRegions: 10,
		UITimeoutMs:           3000,
		Connectivity:          FourConnected,
		UsePendingTimestamp:   false,
		SkipUndefinedResponse: true,
	}
}

func NewInputStateMachineParams(kwargs map[string]interface{}) *InputStateMachineParams {
	params := DefaultInputStateMachineParams()

	if v, ok := kwargs["jank_threshold_ms"]; ok {
		params.JankThresholdMs = int64(v.(int))
	}

	if v, ok := kwargs["jank_filter_value"]; ok {
		switch t := v.(type) {
		case int:
			params.JankFilterValue = float64(t)
		case float64:
			params.JankFilterValue = t
		}
	}

	if v, ok := kwargs["ui_timeout_ms"]; ok {
		params.UITimeoutMs = int64(v.(int))
	}

	if v, ok := kwargs["connectivity"]; ok {
		switch v.(string) {
		case "one":
			params.Connectivity = OneConnected
		case "four":
			params.Connectivity = FourConnected
		case "eight":
			params.Connectivity = EightConnected
		}
	}

	if v, ok := kwargs["local_resp_pct"]; ok {
		switch t := v.(type) {
		case int:
			params.LocalResponsePercent = float64(t)
		case float64:
			params.LocalResponsePercent = t
		}
	}

	if v, ok := kwargs["global_resp_pct"]; ok {
		switch t := v.(type) {
		case int:
			params.GlobalResponsePercent = float64(t)
		case float64:
			params.GlobalResponsePercent = t
		}
	}

	if v, ok := kwargs["global_regions"]; ok {
		params.GlobalResponseRegions = v.(int)
	}

	if v, ok := kwargs["local_regions"]; ok {
		params.LocalResponseRegions = v.(int)
	}

	if v, ok := kwargs["use_pending_ts"]; ok {
		params.UsePendingTimestamp = v.(bool)
	}

	if v, ok := kwargs["skip_undefined_resp"]; ok {
		params.SkipUndefinedResponse = v.(bool)
	}

	if v, ok := kwargs["jank_on_frame_update"]; ok {
		params.JankOnFrameUpdate = v.(bool)
	}

	fmt.Println("ISM Parameters:", *params)

	return params
}

// InputStateMachine states
const (
	InputStateWaitInput = iota
	InputStateWaitResponse
	InputStateMeasureLocal
	InputStateMeasureGlobal
	InputStateMeasureScroll
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
	curResult *InputEventResult
	curEvent  *TouchScreenEvent

	pendingResponseStartNs int64
	scrollKeepaliveNs      int64
}

// Create a new InputStateMachine with the default parameters.
func NewInputStateMachine() *InputStateMachine {
	return &InputStateMachine{
		Params:                 DefaultInputStateMachineParams(),
		pendingResponseStartNs: InvalidResponseTime,
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
	InvalidResponseTime     = -1
	InvalidResponseDuration = 999999999999
)

const (
	TapEventFinishTimeout = iota
	TapEventFinishShortCircuit
)

// InputEventResult encapsulates the response detail and performance metrics of a
// single tap event.
type InputEventResult struct {
	TimestampNs    int64           `json:"timestamp_ns"`
	FinishNs       int64           `json:"finish_ns"`
	FinishType     int             `json:"finish_type"`
	LocalResponse  *ResponseDetail `json:"local_response"`
	GlobalResponse *ResponseDetail `json:"global_response"`
	Jank           []*JankEvent    `json:"jank_events"`

	EventType    int   `json:"event_type"`
	ScrollStopNs int64 `json:"scroll_stop_ns"`

	prevFrameTimeNs int64
}

func NewInputEventResult(event *TouchScreenEvent) *InputEventResult {
	return &InputEventResult{
		EventType:      event.What,
		TimestampNs:    event.Timestamp,
		LocalResponse:  NewResponseDetail(),
		GlobalResponse: NewResponseDetail(),
		Jank:           make([]*JankEvent, 0),
	}
}

func (t *InputEventResult) HasLocalResponse() bool {
	return t.LocalResponse.HasResponse()
}

func (t *InputEventResult) HasGlobalResponse() bool {
	return t.GlobalResponse.HasResponse()
}

func (t *InputEventResult) HasResponse() bool {
	return t.HasLocalResponse() || t.HasLocalResponse()
}

func (t *InputEventResult) TouchResponseMs() int64 {
	if t.HasLocalResponse() {
		return (t.LocalResponse.StartNs - t.TimestampNs) / nsPerMs
	} else {
		return t.GlobalResponseMs()
	}
}

func (t *InputEventResult) GlobalResponseMs() int64 {
	if t.HasGlobalResponse() {
		return (t.GlobalResponse.StartNs - t.TimestampNs) / nsPerMs
	} else {
		return InvalidResponseDuration
	}
}

func (t *InputEventResult) LocalResponseDurationMs() int64 {
	return t.LocalResponse.DurationMs()
}

func (t *InputEventResult) GlobalResponseDurationMs() int64 {
	return t.GlobalResponse.DurationMs()
}

func (t *InputEventResult) ContentDelayMs() int64 {
	if t.HasGlobalResponse() {
		return (t.GlobalResponse.StartNs - t.TimestampNs) / nsPerMs
	} else {
		return InvalidResponseDuration
	}
}

// ResponseDetail contains the information needed to capture the performance
// metrics for either a local or global response to an input event.
type ResponseDetail struct {
	StartNs int64 `json:"start_ns"`
	EndNs   int64 `json:"end_ns"`

	params *InputStateMachineParams `json:"-"`
}

func NewResponseDetail() *ResponseDetail {
	return &ResponseDetail{
		StartNs: InvalidResponseTime,
		EndNs:   InvalidResponseTime,
	}
}

func (response *ResponseDetail) HasResponse() bool {
	return response.StartNs != InvalidResponseTime
}

func (response *ResponseDetail) DurationMs() int64 {
	if response.StartNs == InvalidResponseTime || response.EndNs == InvalidResponseTime {
		return InvalidResponseDuration
	} else {
		return (response.EndNs - response.StartNs) / nsPerMs
	}
}

func (response *ResponseDetail) DurationNs() int64 {
	if response.StartNs == InvalidResponseTime || response.EndNs == InvalidResponseTime {
		return InvalidResponseDuration
	} else {
		return response.EndNs - response.StartNs
	}
}

// Update the response with a new diff sample.
func (response *ResponseDetail) onFrameDiff(diff *FrameDiffSample) {
	if diff.PctDiff == 0.0 {
		// Don't advance the state at all.
		return
	}

	// Advance the end time for this response
	response.EndNs = diff.TimestampNs()
}

// Short-circuit the current result/analysis. This gets called when we're
// analyzing the post-tap stream and another input event comes along.
func (ism *InputStateMachine) shortCircuit(ts int64) *InputEventResult {
	if ism.curResult == nil {
		return nil
	}

	// Finish detail
	ism.curResult.FinishType = TapEventFinishShortCircuit
	ism.curResult.FinishNs = ts
	ism.curResult.prevFrameTimeNs = 0

	// TODO: Do we need to detect timeouts here?
	// If the diffs interlace zeros, then probably not since we'll have
	// frequent timestamps.

	return ism.curResult
}

// State change InputStateWaitResponse --> InputStateWaitInput
func (ism *InputStateMachine) handleTimeout(ts int64) *InputEventResult {
	res := ism.curResult
	res.FinishType = TapEventFinishTimeout
	res.FinishNs = ts
	res.prevFrameTimeNs = 0

	// --> InputStateWaitInput
	ism.reset()

	return res
}

// Reset to the start state (--> InputStateWaitInput).
func (ism *InputStateMachine) reset() {
	if ismDebug {
		fmt.Println("State = wait input")
	}

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

		if ismDebug {
			fmt.Println("Local ratio:", ratio)
		}

		if ratio >= ism.Params.LocalResponsePercent &&
			(ism.Params.LocalResponseRegions == 0 || len(diff.GridEntries) <= ism.Params.LocalResponseRegions) {
			return responseTypeLocal
		}
	} else if ismDebug {
		fmt.Println("No local diff, pct=", diff.PctDiff)
	}

	// Not a local diff
	if diff.PctDiff >= ism.Params.GlobalResponsePercent || len(diff.GridEntries) >= ism.Params.GlobalResponseRegions {
		return responseTypeGlobal
	}

	return responseTypeNeither
}

// Update state and possibly return an event result
func (ism *InputStateMachine) OnTouchEvent(event *TouchScreenEvent) *InputEventResult {

	// For all other touch events, this short-circuits the current state if
	// we're not at the start/wait state.
	var cur *InputEventResult = nil

	// Skip key events, except power
	if event.What == TouchScreenEventKey {
		if event.Code == KEYCODE_POWER {
			// Hard key, we need this one.
			// TODO: Should we just look at the screen on/off logs?
			if ism.curState != InputStateWaitInput {
				cur = ism.shortCircuit(event.Timestamp)
			}
			ism.reset()
			return cur
		}
		return nil
	}

	// New event staring
	if event.What == TouchScreenEventTap || event.What == TouchScreenEventScrollStart {
		if ism.curState != InputStateWaitInput {
			cur = ism.shortCircuit(event.Timestamp)
		}
		// Clear state
		ism.reset()

		// If it's a tap or scroll event, transition --> InputStateWaitResponse
		ism.startWaitingForResponse(event)
	} else if event.What == TouchScreenEventScrollEnd && ism.curEvent != nil {
		// No state transition though, we want to keep evaluating the output.
		// TODO: Eventually, this might want to transition to a new scrolling
		// non-active state.
		ism.curResult.ScrollStopNs = event.Timestamp
	}

	return cur
}

// State change from InputStateWaitInput --> InputStateWaitResponse
func (ism *InputStateMachine) startWaitingForResponse(event *TouchScreenEvent) {
	if ismDebug {
		fmt.Println("State = wait response")
	}

	ism.curState = InputStateWaitResponse
	ism.curEvent = event
	ism.curResult = NewInputEventResult(event)
	//ism.curResult.prevFrameTime = event.Timestamp / 1000000
}

// State change from InputStateWaitResponse --> InputStateMeasureLocal
func (ism *InputStateMachine) startMeasuringLocalRepsonse(diff *FrameDiffSample) {
	if ismDebug {
		fmt.Println("State = measure local")
	}

	ism.curState = InputStateMeasureLocal
	ism.startMeasuringRepsonse(ism.curResult.LocalResponse, diff)
}

// State change from InputStateWaitResponse or InputStateMeasureLocal --> InputStateMeasureGlobal
func (ism *InputStateMachine) startMeasuringGlobalRepsonse(diff *FrameDiffSample) {
	if ismDebug {
		fmt.Println("State = measure global")
	}

	ism.curState = InputStateMeasureGlobal
	ism.startMeasuringRepsonse(ism.curResult.GlobalResponse, diff)
}

func (ism *InputStateMachine) startMeasuringRepsonse(response *ResponseDetail, diff *FrameDiffSample) {
	if ism.pendingResponseStartNs != InvalidResponseTime {
		// Inch closer to the actual start by marking the start at the first
		// refresh event we saw while waiting
		response.StartNs = ism.pendingResponseStartNs
		ism.pendingResponseStartNs = InvalidResponseTime
	} else {
		response.StartNs = diff.TimestampNs()
	}

	response.EndNs = diff.TimestampNs()
}

func (ism *InputStateMachine) checkJank(timestampNs int64) {
	if ism.curResult != nil {
		if ism.curResult.prevFrameTimeNs > 0 {
			delta := (timestampNs - ism.curResult.prevFrameTimeNs) / 1000000
			if delta >= ism.Params.JankThresholdMs {
				ism.curResult.Jank = append(ism.curResult.Jank, &JankEvent{
					TimestampNs: timestampNs,
					JankAmount:  delta,
				})
			}
		}
		ism.curResult.prevFrameTimeNs = timestampNs
	}
}

// Update state and possibly return an event result.
func (ism *InputStateMachine) OnFrameDiff(diff *FrameDiffSample) *InputEventResult {

	// Short circuit: we don't do anything with diffs if we're waiting for
	// input.
	if ism.curState == InputStateWaitInput {
		return nil
	}

	// We have a response (which could be 0.0%), but what type is it?
	rt := ism.getResponseType(diff)

	// Reset the pending response start if the diff is actully zero
	if ism.Params.UsePendingTimestamp &&
		rt == responseTypeNone && ism.curState == InputStateWaitResponse {

		ism.pendingResponseStartNs = InvalidResponseTime
	}

	// Jank check - applies to taps and scrolls, all states unless waiting for input.
	if ism.curResult != nil && diff.PctDiff > ism.Params.JankFilterValue &&
		!ism.Params.JankOnFrameUpdate {

		ism.checkJank(diff.TimestampNs())
	}

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
		case InputStateMeasureScroll:
			ns = ism.scrollKeepaliveNs
		}

		if ns > 0 && (diff.TimestampNs()-ns)/nsPerMs >= ism.Params.UITimeoutMs {
			// We timed out
			return ism.handleTimeout(diff.TimestampNs())
		} else {
			// Otherwise, no change
			return nil
		}
	}

	if ismDebug {
		fmt.Println("Diff TS =", diff.Timestamp)
	}

	// OK, we have a resonse of some sort and we're not in the start state.
	// We'll either update the current response, or transition to a different
	// start.

	switch ism.curState {
	default:
		{
			panic(fmt.Sprint("Unexpected state: %v", ism.curState))
		}
	case InputStateWaitResponse:
		{
			if ism.curEvent.What == TouchScreenEventScrollStart {
				// Transition to the
				return nil
			} else if ism.Params.SkipUndefinedResponse && rt == responseTypeNeither {
				// No change
				if ismDebug {
					fmt.Println("Skipping non-local/non-global response while waiting")
				}

				return nil
			} else if rt == responseTypeLocal {
				if ismDebug {
					fmt.Println("Local response (wait response)")
				}
				// State transition to wait start measuring local response
				ism.startMeasuringLocalRepsonse(diff)
			} else {
				if ismDebug {
					fmt.Println("Global response (wait response)")
				}
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
				if ismDebug {
					fmt.Println("Local response (measure local)")
				}
				// TODO: This is a simplified -- we're considering any
				// Keep measuring local
				ism.curResult.LocalResponse.onFrameDiff(diff)
			} else {
				if ismDebug {
					fmt.Println("Global response (measure local)")
				}
				// State transition to start measuring the global response
				ism.startMeasuringGlobalRepsonse(diff)
			}
		}
	case InputStateMeasureGlobal:
		{
			if ismDebug {
				fmt.Println("? response (measure global)")
			}
			// Even if the diff is only local, we had a global response and we
			// don't flip-flop states.
			ism.curResult.GlobalResponse.onFrameDiff(diff)
		}
	case InputStateMeasureScroll:
		{
			// All we need to do is update the keepalive for timeouts. The jank
			// detection is already handled.
			ism.scrollKeepaliveNs = diff.TimestampNs()
		}
	}

	return nil
}

func (ism *InputStateMachine) OnFrameRefresh(event *FrameRefreshEvent) *InputEventResult {
	// At this point, we'll only use this info to update the jankiness, so we'll always
	// return nil.

	// It's an option where to do this.
	if !ism.Params.JankOnFrameUpdate {
		return nil
	}

	switch ism.curState {
	default:
		{
			return nil
		}
	case InputStateWaitResponse:
		{
			if ism.Params.UsePendingTimestamp {
				if ism.pendingResponseStartNs == InvalidResponseTime {
					ism.pendingResponseStartNs = event.SysTimeNs
				}
			}
		}
		fallthrough
	case InputStateMeasureScroll:
		fallthrough
	case InputStateMeasureLocal:
		fallthrough
	case InputStateMeasureGlobal:
		{
			// Jank check - applies to taps and scrolls, all states unless waiting for input.
			if ism.curResult != nil {
				ism.checkJank(event.SysTimeNs)
			}
		}
	}
	return nil
}

// Called when the input log stream is finished
func (ism *InputStateMachine) Finish(ts int64) *InputEventResult {
	if ism.curState != InputStateWaitInput {
		// Just short-circuit the current test in the same way as if we would
		// received a new touch event.
		return ism.shortCircuit(ts)
	}
	return nil
}

type InputStateMachineProcessor struct {
	Source phonelab.Processor
	Args   map[string]interface{}
}

func (proc *InputStateMachineProcessor) Process() <-chan interface{} {

	outChan := make(chan interface{})
	inChan := proc.Source.Process()

	go func() {
		ism := NewInputStateMachine()

		if len(proc.Args) > 0 {
			ism.Params = NewInputStateMachineParams(proc.Args)
		}

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

////////////////////////////////////////////////////////////////////////////////

func GenerateISMProcessor(source *phonelab.PipelineSourceInstance,
	kwargs map[string]interface{}) phonelab.Processor {

	return &InputStateMachineProcessor{
		Source: source.Processor,
		Args:   kwargs,
	}
}
