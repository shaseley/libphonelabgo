package libphonelabgo

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func TestParseIFMotionEvent(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	payload := `{"msg":"dispatch","ts":64159277303000,"dev":5,"src":4098,"pflags":1644167168,"action":0,"a_btn":0,"flags":0,"meta":0,"btn_state":0,"eflags":0,"x_p":1.000000,"y_p":1.000000,"dtime":64159277303000,"ptrs":[{"id":0,"tool":1,"x":836.000000,"y":2279.000000,"pr":0.337500,"sz":0.003922,"tch_mj":41.476997,"tch_mn":41.476997,"tl_mj":41.476997,"tl_mn":41.476997,"orient":0.000000}]}`

	parser := NewIFMotionEventParser()
	require.NotNil(parser)

	log, err := parser.Parse(payload)
	assert.Nil(err)
	assert.NotNil(log)
	typedLog, ok := log.(*IFMotionEventLog)
	require.True(ok)

	expected := &IFMotionEventLog{
		Msg:          "dispatch",
		Timestamp:    64159277303000,
		DeviceId:     5,
		Source:       4098,
		PolicyFlags:  1644167168,
		Action:       0,
		ActionButton: 0,
		Flags:        0,
		MetaState:    0,
		ButtonState:  0,
		EdgeFlags:    0,
		XPrecision:   1.0,
		YPrecision:   1.0,
		Downtime:     64159277303000,
		PointerData: []*IFPointerData{
			&IFPointerData{
				Id:          0,
				ToolType:    1,
				XPos:        836.0,
				YPos:        2279.0,
				Pressure:    0.337500,
				Size:        0.003922,
				TouchMajor:  41.476997,
				TouchMinor:  41.476997,
				ToolMajor:   41.476997,
				ToolMinor:   41.476997,
				Orientation: 0.0,
			},
		},
	}

	require.True(reflect.DeepEqual(expected, typedLog))
}

func TestParseIFKeyEvent(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	payload := `{"msg":"dispatch","ts":64162858851000,"dev":3,"src":257,"pflags":1644167168,"action":0,"flags":8,"key_code":25,"scan_code":114,"meta":0,"repeat":0,"dtime":64162858851000}`

	parser := NewIFKeyEventParser()
	require.NotNil(parser)

	log, err := parser.Parse(payload)
	assert.Nil(err)
	assert.NotNil(log)
	typedLog, ok := log.(*IFKeyEventLog)
	require.True(ok)

	expected := &IFKeyEventLog{
		Msg:         "dispatch",
		Timestamp:   64162858851000,
		DeviceId:    3,
		Source:      257,
		PolicyFlags: 1644167168,
		Action:      0,
		Flags:       8,
		KeyCode:     25,
		ScanCode:    114,
		MetaState:   0,
		RepeatCount: 0,
		Downtime:    64162858851000,
	}

	require.True(reflect.DeepEqual(expected, typedLog))
}

func TestActionMasked(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	tests := []struct {
		action   int
		expected int
	}{
		{ACTION_DOWN, ACTION_DOWN},
		{ACTION_UP, ACTION_UP},
		{ACTION_MOVE, ACTION_MOVE},
		{ACTION_CANCEL, ACTION_CANCEL},
		{ACTION_POINTER_1_UP, ACTION_POINTER_UP},
		{ACTION_POINTER_2_UP, ACTION_POINTER_UP},
		{ACTION_POINTER_3_UP, ACTION_POINTER_UP},
		{ACTION_POINTER_1_DOWN, ACTION_POINTER_DOWN},
		{ACTION_POINTER_2_DOWN, ACTION_POINTER_DOWN},
		{ACTION_POINTER_3_DOWN, ACTION_POINTER_DOWN},
	}

	for _, test := range tests {
		event := &IFMotionEventLog{
			Action: test.action,
		}
		res := event.GetMaskedAction()
		assert.Equal(test.expected, res)
	}
}

func TestActionPointerIndex(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	tests := []struct {
		action   int
		expected int
	}{
		{ACTION_POINTER_DOWN, 0},
		{ACTION_POINTER_UP, 0},
		{ACTION_UP, 0},
		{ACTION_MOVE, 0},
		{ACTION_CANCEL, 0},
		{ACTION_POINTER_1_UP, 0},
		{ACTION_POINTER_2_UP, 1},
		{ACTION_POINTER_3_UP, 2},
		{ACTION_POINTER_1_DOWN, 0},
		{ACTION_POINTER_2_DOWN, 1},
		{ACTION_POINTER_3_DOWN, 2},
	}

	for _, test := range tests {
		event := &IFMotionEventLog{
			Action: test.action,
		}
		res := event.GetActionIndex()
		assert.Equal(test.expected, res)
	}
}
