package libphonelabgo

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInputGestureSimpleTap(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	detector := NewGestureDetector(TouchSlopScaled)

	down := &IFMotionEventLog{
		Action: ACTION_DOWN,
		PointerData: []*IFPointerData{
			&IFPointerData{
				Id:   0,
				XPos: 700.0,
				YPos: 200.0,
			},
		},
	}

	move1 := &IFMotionEventLog{
		Action: ACTION_MOVE,
		PointerData: []*IFPointerData{
			&IFPointerData{
				Id:   0,
				XPos: 701.0,
				YPos: 201.0,
			},
		},
	}

	move2 := &IFMotionEventLog{
		Action: ACTION_MOVE,
		PointerData: []*IFPointerData{
			&IFPointerData{
				Id:   0,
				XPos: 702.0,
				YPos: 202.0,
			},
		},
	}

	up := &IFMotionEventLog{
		Action: ACTION_UP,
		PointerData: []*IFPointerData{
			&IFPointerData{
				Id:   0,
				XPos: 702.0,
				YPos: 202.0,
			},
		},
	}

	var err error
	var event *TouchScreenEvent

	// Bogus trace timestamp
	ts := float64(0)

	assert.Equal(GestureStateNone, detector.State())

	event, err = detector.OnTouchEvent(ts, down)
	assert.Equal(GestureStateTapping, detector.State())
	assert.Nil(event)
	assert.Nil(err)

	event, err = detector.OnTouchEvent(ts, move1)
	assert.Equal(GestureStateTapping, detector.State())
	assert.Nil(event)
	assert.Nil(err)

	event, err = detector.OnTouchEvent(ts, move2)
	assert.Equal(GestureStateTapping, detector.State())
	assert.Nil(event)
	assert.Nil(err)

	event, err = detector.OnTouchEvent(ts, up)
	assert.Equal(GestureStateNone, detector.State())
	assert.Nil(err)

	require.NotNil(event)
	assert.Equal(TouchScreenEventTap, event.What)
}

func TestInputGestureSimpleScroll(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	detector := NewGestureDetector(TouchSlopScaled)

	down := &IFMotionEventLog{
		Action: ACTION_DOWN,
		PointerData: []*IFPointerData{
			&IFPointerData{
				Id:   0,
				XPos: 700.0,
				YPos: 200.0,
			},
		},
	}

	moves := []*IFMotionEventLog{
		&IFMotionEventLog{
			Action: ACTION_MOVE,
			PointerData: []*IFPointerData{
				&IFPointerData{
					Id:   0,
					XPos: 701.0,
					YPos: 201.0,
				},
			},
		},
		&IFMotionEventLog{
			Action: ACTION_MOVE,
			PointerData: []*IFPointerData{
				&IFPointerData{
					Id:   0,
					XPos: 702.0,
					YPos: 222.0,
				},
			},
		},
		&IFMotionEventLog{
			Action: ACTION_MOVE,
			PointerData: []*IFPointerData{
				&IFPointerData{
					Id:   0,
					XPos: 702.0,
					YPos: 242.0,
				},
			},
		},
		&IFMotionEventLog{
			Action: ACTION_MOVE,
			PointerData: []*IFPointerData{
				&IFPointerData{
					Id:   0,
					XPos: 702.0,
					YPos: 262.0,
				},
			},
		},
	}

	scrollIndex := 2

	up := &IFMotionEventLog{
		Action: ACTION_UP,
		PointerData: []*IFPointerData{
			&IFPointerData{
				Id: 0,
			},
		},
	}

	var err error
	var event *TouchScreenEvent

	// Bogus trace timestamp
	ts := float64(0)

	assert.Equal(GestureStateNone, detector.State())

	event, err = detector.OnTouchEvent(ts, down)
	assert.Equal(GestureStateTapping, detector.State())
	assert.Nil(event)
	assert.Nil(err)

	for i, move := range moves {
		event, err = detector.OnTouchEvent(ts, move)
		assert.Nil(err)

		if i < scrollIndex {
			assert.Equal(GestureStateTapping, detector.State())
			assert.Nil(event)
		} else if i == scrollIndex {
			assert.Equal(GestureStateScrolling, detector.State())
			require.NotNil(event)
			assert.Equal(TouchScreenEventScrollStart, event.What)
		} else {
			assert.Equal(GestureStateScrolling, detector.State())
			require.NotNil(event)
			assert.Equal(TouchScreenEventScroll, event.What)
		}
	}

	event, err = detector.OnTouchEvent(ts, up)
	assert.Equal(GestureStateNone, detector.State())
	assert.Nil(err)

	require.NotNil(event)
	assert.Equal(TouchScreenEventScrollEnd, event.What)
}
