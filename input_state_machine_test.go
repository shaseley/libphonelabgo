package libphonelabgo

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func commonTestInputStateMachine(events []*TouchScreenEvent, diffStream []*FrameDiffSample,
	states []int, expected *TapEventResult, t *testing.T) {

	assert := assert.New(t)
	require := require.New(t)

	require.True(len(events) == 1 || len(events) == 2)

	// We check the state for each diff
	require.Equal(len(states), len(diffStream))

	ism := NewInputStateMachine()

	require.NotNil(ism)
	assert.Equal(InputStateWaitInput, ism.curState)

	res := ism.OnTouchEvent(events[0])
	assert.Equal(InputStateWaitResponse, ism.curState)
	assert.Nil(res)

	// Now, process the diffs
	for i, diff := range diffStream {

		diff.initScreenGrid(allScreenGrids[0])

		res = ism.OnFrameDiff(diff)
		assert.Equal(states[i], ism.curState)

		if res != nil {
			break
		}
	}

	if res == nil {
		require.Equal(2, len(events))
		res = ism.OnTouchEvent(events[1])
	}

	require.NotNil(res)

	// Finally, check the expected result
	assert.True(reflect.DeepEqual(expected, res))
}

func TestISMShortCircuit(t *testing.T) {

	events := []*TouchScreenEvent{
		&TouchScreenEvent{
			What:      TouchScreenEventTap,
			Timestamp: 100 * nsPerMs,
		},
		&TouchScreenEvent{
			What:      TouchScreenEventScroll,
			Timestamp: 500 * nsPerMs,
		},
	}

	expected := &TapEventResult{
		TimestampNs: events[0].Timestamp,
		FinishNs:    events[1].Timestamp,
		FinishType:  TapEventFinishShortCircuit,
		LocalResponse: &ResponseDetail{
			Jank:    make([]*JankEvent, 0),
			StartNs: InvalidResponseTime,
			EndNs:   InvalidResponseTime,
		},
		GlobalResponse: &ResponseDetail{
			Jank:    make([]*JankEvent, 0),
			StartNs: InvalidResponseTime,
			EndNs:   InvalidResponseTime,
		},
	}

	commonTestInputStateMachine(events, nil, nil, expected, t)
}

func TestISMLocalResponse(t *testing.T) {
	// Touch the upper left corner
	events := []*TouchScreenEvent{
		&TouchScreenEvent{
			What:      TouchScreenEventTap,
			Timestamp: 100 * nsPerMs,
		},
	}

	diffs := []*FrameDiffSample{
		// No diff
		&FrameDiffSample{
			SFFrameDiff: SFFrameDiff{
				Timestamp:   150 * nsPerMs,
				PctDiff:     0.0,
				GridWH:      8,
				GridEntries: []*GridEntry{},
			},
		},
		// No diff
		&FrameDiffSample{
			SFFrameDiff: SFFrameDiff{
				Timestamp:   200 * nsPerMs,
				PctDiff:     0.0,
				GridWH:      8,
				GridEntries: []*GridEntry{},
			},
		},
		// No diff
		&FrameDiffSample{
			SFFrameDiff: SFFrameDiff{
				Timestamp:   200 * nsPerMs,
				PctDiff:     0.0,
				GridWH:      8,
				GridEntries: []*GridEntry{},
			},
		},
		// Local response (0, 0), 100% of the diff
		&FrameDiffSample{
			SFFrameDiff: SFFrameDiff{
				Timestamp: 250 * nsPerMs,
				PctDiff:   100.0 / 72.0,
				GridWH:    8,
				GridEntries: []*GridEntry{
					&GridEntry{
						Position: 56,
						Value:    50.0,
					},
				},
			},
		},
		// Same thing
		&FrameDiffSample{
			SFFrameDiff: SFFrameDiff{
				Timestamp: 300 * nsPerMs,
				PctDiff:   100.0 / 72.0,
				GridWH:    8,
				GridEntries: []*GridEntry{
					&GridEntry{
						Position: 56,
						Value:    50.0,
					},
				},
			},
		},
		// Now settle the UI
		// No diff
		&FrameDiffSample{
			SFFrameDiff: SFFrameDiff{
				Timestamp:   3300 * nsPerMs,
				PctDiff:     0.0,
				GridWH:      8,
				GridEntries: []*GridEntry{},
			},
		},
		// This one should trigger the timeout
		&FrameDiffSample{
			SFFrameDiff: SFFrameDiff{
				Timestamp:   6300 * nsPerMs,
				PctDiff:     0.0,
				GridWH:      8,
				GridEntries: []*GridEntry{},
			},
		},
	}
	states := []int{
		InputStateWaitResponse,
		InputStateWaitResponse,
		InputStateWaitResponse,
		InputStateMeasureLocal,
		InputStateMeasureLocal,
		InputStateMeasureLocal,
		InputStateWaitInput,
	}

	expected := &TapEventResult{
		TimestampNs: events[0].Timestamp,
		FinishNs:    6300 * nsPerMs,
		FinishType:  TapEventFinishTimeout,
		LocalResponse: &ResponseDetail{
			Jank:    make([]*JankEvent, 0),
			StartNs: 250 * nsPerMs,
			EndNs:   300 * nsPerMs,
		},
		GlobalResponse: &ResponseDetail{
			Jank:    make([]*JankEvent, 0),
			StartNs: InvalidResponseTime,
			EndNs:   InvalidResponseTime,
		},
	}

	commonTestInputStateMachine(events, diffs, states, expected, t)
}

func TestISMLocalGlobalResponse(t *testing.T) {
	// Touch the upper left corner
	events := []*TouchScreenEvent{
		&TouchScreenEvent{
			What:      TouchScreenEventTap,
			Timestamp: 100 * nsPerMs,
		},
	}

	diffs := []*FrameDiffSample{
		// No diff
		&FrameDiffSample{
			SFFrameDiff: SFFrameDiff{
				Timestamp:   150 * nsPerMs,
				PctDiff:     0.0,
				GridWH:      8,
				GridEntries: []*GridEntry{},
			},
		},
		// No diff
		&FrameDiffSample{
			SFFrameDiff: SFFrameDiff{
				Timestamp:   200 * nsPerMs,
				PctDiff:     0.0,
				GridWH:      8,
				GridEntries: []*GridEntry{},
			},
		},
		// No diff
		&FrameDiffSample{
			SFFrameDiff: SFFrameDiff{
				Timestamp:   200 * nsPerMs,
				PctDiff:     0.0,
				GridWH:      8,
				GridEntries: []*GridEntry{},
			},
		},
		// Local response (0, 0), 100% of the diff
		&FrameDiffSample{
			SFFrameDiff: SFFrameDiff{
				Timestamp: 250 * nsPerMs,
				PctDiff:   100.0 / 72.0,
				GridWH:    8,
				GridEntries: []*GridEntry{
					&GridEntry{
						Position: 56,
						Value:    50.0,
					},
				},
			},
		},
		// Same thing
		&FrameDiffSample{
			SFFrameDiff: SFFrameDiff{
				Timestamp: 300 * nsPerMs,
				PctDiff:   100.0 / 72.0,
				GridWH:    8,
				GridEntries: []*GridEntry{
					&GridEntry{
						Position: 56,
						Value:    50.0,
					},
				},
			},
		},
		// Now settle the UI, but don't timeout
		// No diff
		&FrameDiffSample{
			SFFrameDiff: SFFrameDiff{
				Timestamp:   3300 * nsPerMs,
				PctDiff:     0.0,
				GridWH:      8,
				GridEntries: []*GridEntry{},
			},
		},
		// Now a large diff
		&FrameDiffSample{
			SFFrameDiff: SFFrameDiff{
				Timestamp: 6300 * nsPerMs,
				PctDiff:   (8.0 * 100.0) / 36.0,
				GridWH:    8,
				GridEntries: []*GridEntry{
					&GridEntry{
						Position: 56,
						Value:    100,
					},
					&GridEntry{
						Position: 57,
						Value:    100,
					},
					&GridEntry{
						Position: 58,
						Value:    100,
					},
					&GridEntry{
						Position: 59,
						Value:    100,
					},
					&GridEntry{
						Position: 48,
						Value:    100,
					},
					&GridEntry{
						Position: 49,
						Value:    100,
					},
					&GridEntry{
						Position: 50,
						Value:    100,
					},
					&GridEntry{
						Position: 51,
						Value:    100,
					},
				},
			},
		},
		&FrameDiffSample{
			SFFrameDiff: SFFrameDiff{
				Timestamp: 6400 * nsPerMs,
				PctDiff:   100.0 / 36.0,
				GridWH:    8,
				GridEntries: []*GridEntry{
					&GridEntry{
						Position: 56,
						Value:    100,
					},
				},
			},
		},
		// This will cause the timeout
		&FrameDiffSample{
			SFFrameDiff: SFFrameDiff{
				Timestamp:   11500 * nsPerMs,
				PctDiff:     0.0,
				GridWH:      8,
				GridEntries: []*GridEntry{},
			},
		},
	}
	states := []int{
		InputStateWaitResponse,
		InputStateWaitResponse,
		InputStateWaitResponse,
		InputStateMeasureLocal,
		InputStateMeasureLocal,
		InputStateMeasureLocal,
		InputStateMeasureGlobal,
		InputStateMeasureGlobal,
		InputStateWaitInput,
	}

	expected := &TapEventResult{
		TimestampNs: events[0].Timestamp,
		FinishNs:    11500 * nsPerMs,
		FinishType:  TapEventFinishTimeout,
		LocalResponse: &ResponseDetail{
			Jank:    make([]*JankEvent, 0),
			StartNs: 250 * nsPerMs,
			EndNs:   300 * nsPerMs,
		},
		GlobalResponse: &ResponseDetail{
			Jank:    make([]*JankEvent, 0),
			StartNs: 6300 * nsPerMs,
			EndNs:   6400 * nsPerMs,
		},
	}

	commonTestInputStateMachine(events, diffs, states, expected, t)
}
