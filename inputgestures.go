package libphonelabgo

import (
	"errors"
	"fmt"
	"math"
)

type TouchScreenEvent struct {
	What      int
	Timestamp int64
	TraceTime float64
	X         float64
	Y         float64
	Code      int
}

func (event *TouchScreenEvent) MonotonicTimestamp() float64 {
	return event.TraceTime
}

// To start with, we only detect scrolls and taps. Further, we don't make any
// distinction between short and long taps. In the future, we may want to add
// more gestures like long tap, double tap, and fling.
const (
	TouchScreenEventKey = iota
	TouchScreenEventTap
	TouchScreenEventScrollStart
	TouchScreenEventScroll
	TouchScreenEventScrollEnd
)

type GestureDetector struct {
	TouchSlop int

	// internal state
	touchSlopSquare float64
	state           int
	downFocusX      float64
	lastFocusX      float64
	downFocusY      float64
	lastFocusY      float64
	initialDowntime int64
	pointerState    GesturePointerState
}

const (
	GestureStateNone = iota
	GestureStateTapping
	GestureStateScrolling
)

type PointerState struct {
	X float64
	Y float64
}

type GesturePointerState map[int]*PointerState

func (g GesturePointerState) Update(event *IFMotionEventLog) error {
	maskedAction := event.GetMaskedAction()
	pointerIndex := event.GetActionIndex()

	switch maskedAction {
	case ACTION_UP:
		{
			// Remove all pointers
			g.Clear()
		}
	case ACTION_POINTER_UP:
		{
			// Remove a single pointer
			delete(g, pointerIndex)
		}
	case ACTION_DOWN:
		fallthrough
	case ACTION_POINTER_DOWN:
		{
			// Start tracking the pointer
			g[pointerIndex] = &PointerState{
				X: -1.0,
				Y: -1.0,
			}
			// We also want to update any pointer positions
		}
		fallthrough
	case ACTION_MOVE:
		{
			// Update all existing pointers
			for _, ptr := range event.PointerData {
				if data, ok := g[ptr.Id]; !ok {
					return fmt.Errorf("Not tracking pointer id: %v", ptr.Id)
				} else {
					data.X = ptr.XPos
					data.Y = ptr.YPos
				}
			}
		}
	}

	return nil
}

func (g GesturePointerState) Clear() {
	for key := range g {
		delete(g, key)
	}
}

// Get the focus point of all the pointers that are down. If no pointers are
// down, this returns (-1, -1). If only a single pointer is down, this returns
// (X, Y) of the last known location. Otherwise, this averages the X and Y
// coordinates of all last known pointer locations.
func (g GesturePointerState) GetFocus() (focusX, focusY float64) {

	if len(g) == 0 {
		focusY, focusY = -1.0, -1.0
		return
	}

	focusY, focusY = 0.0, 0.0

	for _, state := range g {
		focusX += state.X
		focusY += state.Y
	}

	focusX /= float64(len(g))
	focusY /= float64(len(g))

	return
}

func NewGestureDetector(touchSlop int) *GestureDetector {
	return &GestureDetector{
		TouchSlop:       touchSlop,
		state:           GestureStateNone,
		touchSlopSquare: float64(touchSlop) * float64(touchSlop),
		pointerState:    make(GesturePointerState),
		initialDowntime: 0,
	}
}

func (detector *GestureDetector) OnTouchEvent(tracetime float64, event *IFMotionEventLog) (*TouchScreenEvent, error) {

	if err := detector.pointerState.Update(event); err != nil {
		return nil, err
	}

	// Events on non-primary pointers have the pointer id baked in with the
	// action.
	maskedAction := event.GetMaskedAction()

	// Get the focus position of all pointers
	focusX, focusY := detector.pointerState.GetFocus()

	switch maskedAction {
	case ACTION_POINTER_DOWN:
		fallthrough
	case ACTION_POINTER_UP:
		fallthrough
	case ACTION_DOWN:
		detector.downFocusX = focusX
		detector.lastFocusX = focusX
		detector.downFocusY = focusY
		detector.lastFocusY = focusY

		if maskedAction == ACTION_DOWN {
			// State change - start tap (or scroll). We won't
			// send an event, though
			detector.initialDowntime = event.Timestamp
			detector.state = GestureStateTapping
		}

	case ACTION_MOVE:
		{
			scrollX := detector.lastFocusX - focusX
			scrollY := detector.lastFocusY - focusY

			switch detector.state {
			default:
				{
					detector.Cancel()
					return nil, errors.New("Received ACTION_MOVE while in empty state")
				}
			case GestureStateScrolling:
				{
					// Continue scrolling, if we actually moved.
					if (math.Abs(scrollX) >= 1.0) || (math.Abs(scrollY) >= 1.0) {
						// TODO: send scroll amounts
						outEvent := detector.GenerateTouchScreenEvent(TouchScreenEventScroll, tracetime, event)
						detector.lastFocusX = focusX
						detector.lastFocusY = focusY
						return outEvent, nil
					}
				}
			case GestureStateTapping:
				{
					// Check if we're still close to the down event(s)
					deltaX := focusX - detector.downFocusX
					deltaY := focusY - detector.downFocusY
					distance := (deltaX * deltaX) + (deltaY * deltaY)
					if distance > detector.touchSlopSquare {
						// Start scrolling
						detector.state = GestureStateScrolling
						outEvent := detector.GenerateTouchScreenEvent(TouchScreenEventScrollStart, tracetime, event)
						detector.lastFocusX = focusX
						detector.lastFocusY = focusY
						return outEvent, nil
					}
				}
			}
		}

	case ACTION_UP:
		{
			eventWhat := 0

			switch detector.state {
			default:
				detector.Cancel()
				return nil, errors.New("Received ACTION_UP while in empty state")
			case GestureStateTapping:
				eventWhat = TouchScreenEventTap
			case GestureStateScrolling:
				eventWhat = TouchScreenEventScrollEnd
			}

			outEvent := detector.GenerateTouchScreenEvent(eventWhat, tracetime, event)
			detector.Cancel()
			return outEvent, nil
		}

	case ACTION_CANCEL:
		{
			detector.Cancel()
		}
	}

	return nil, nil
}

func (detector *GestureDetector) GenerateTouchScreenEvent(what int, tracetime float64,
	event *IFMotionEventLog) *TouchScreenEvent {

	return &TouchScreenEvent{
		What:      what,
		Timestamp: event.Timestamp,
		TraceTime: tracetime,
		X:         detector.lastFocusX,
		Y:         detector.lastFocusY,
	}
}

func (detector *GestureDetector) State() int {
	return detector.state
}

func (detector *GestureDetector) Cancel() {
	detector.state = GestureStateNone
	detector.downFocusX = 0.0
	detector.lastFocusX = 0.0
	detector.downFocusY = 0.0
	detector.lastFocusY = 0.0

	detector.pointerState.Clear()
}
