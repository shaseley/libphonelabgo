package libphonelabgo

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func TestParseSFFpsPayload(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	payload := `{"fps":37.8, "tot_frames":84729, "prev_frames":84691, "cur_time": 184538788137586, "prev_time": 184537784018732}`

	parser := NewSurfaceFlingerParser()
	require.NotNil(parser)

	log, err := parser.Parse(payload)
	assert.Nil(err)
	assert.NotNil(log)
	fps, ok := log.(*SFFpsLog)
	require.True(ok)

	expected := &SFFpsLog{
		FPS:           37.8,
		TotalFrames:   84729,
		PrevFrames:    84691,
		SysTimestamp:  184538788137586,
		PrevTimestamp: 184537784018732,
	}

	require.True(reflect.DeepEqual(expected, fps))
}

func TestParseSFFpsPayloadError(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	payload := `{"fps":37.8, "`

	parser := NewSurfaceFlingerParser()
	require.NotNil(parser)

	_, err := parser.Parse(payload)
	assert.NotNil(err)
}

func TestParseSFFrameDiffsOld(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	payload := `{"token":1699, "diffs":[[184533085, 0.000000],[184533941, 0.493028],[184533994, 69.184029],[184534044, 100.000000],[184534146, 99.305557]]}`

	parser := NewSurfaceFlingerParser()
	require.NotNil(parser)

	log, err := parser.Parse(payload)
	assert.Nil(err)
	assert.NotNil(log)
	diffLog, ok := log.(*SFFrameDiffLog)
	require.True(ok)

	expected := &SFFrameDiffLog{
		Token: 1699,
		Diffs: []*SFFrameDiff{
			&SFFrameDiff{
				Timestamp: 184533085,
				PctDiff:   0.0,
				Mode:      0,
				HasColor:  0,
			},
			&SFFrameDiff{
				Timestamp: 184533941,
				PctDiff:   0.493028,
				Mode:      0,
				HasColor:  0,
			},
			&SFFrameDiff{
				Timestamp: 184533994,
				PctDiff:   69.184029,
				Mode:      0,
				HasColor:  0,
			},
			&SFFrameDiff{
				Timestamp: 184534044,
				PctDiff:   100.0,
				Mode:      0,
				HasColor:  0,
			},
			&SFFrameDiff{
				Timestamp: 184534146,
				PctDiff:   99.305557,
				Mode:      0,
				HasColor:  0,
			},
		},
	}

	require.True(reflect.DeepEqual(expected, diffLog))
}

func TestParseSFFrameDiffsOldErr(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	payload := `{"token":1699, "diffs":[[184533085, 0.000000],[184533941, 0.493028],[184533994, 69.184029],[184534044, 100.`

	parser := NewSurfaceFlingerParser()
	require.NotNil(parser)

	_, err := parser.Parse(payload)
	assert.NotNil(err)

	// Too many fields
	payload = `{"token":1699, "diffs":[[184533085, 0.000000, "foo"],[184533941, 0.493028],[184533994, 69.184029]]}`

	_, err = parser.Parse(payload)
	assert.NotNil(err)
}

func TestParseSFFrameDiffsNew(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	payload := `{"token":25, "diffs":[{"ts":30052602, "diff":0.010173, "mode": -1, "color": 0},{"ts":30052668, "diff":0.009833, "mode": -1, "color": 0},{"ts":30052752, "diff":0.008986, "mode": -1, "color": 0},{"ts":30052810, "diff":0.009833, "mode": -1, "color": 0},{"ts":30052973, "diff":0.009664, "mode": -1, "color": 0}]}`

	parser := NewSurfaceFlingerParser()
	require.NotNil(parser)

	log, err := parser.Parse(payload)
	assert.Nil(err)
	assert.NotNil(log)
	diffLog, ok := log.(*SFFrameDiffLog)
	require.True(ok)

	expected := &SFFrameDiffLog{
		Token: 25,
		Diffs: []*SFFrameDiff{
			&SFFrameDiff{
				Timestamp: 30052602,
				PctDiff:   0.010173,
				Mode:      -1,
				HasColor:  0,
			},
			&SFFrameDiff{
				Timestamp: 30052668,
				PctDiff:   0.009833,
				Mode:      -1,
				HasColor:  0,
			},
			&SFFrameDiff{
				Timestamp: 30052752,
				PctDiff:   0.008986,
				Mode:      -1,
				HasColor:  0,
			},
			&SFFrameDiff{
				Timestamp: 30052810,
				PctDiff:   0.009833,
				Mode:      -1,
				HasColor:  0,
			},
			&SFFrameDiff{
				Timestamp: 30052973,
				PctDiff:   0.009664,
				Mode:      -1,
				HasColor:  0,
			},
		},
	}

	require.True(reflect.DeepEqual(expected, diffLog))
}

func TestParseSFFrameDiffsNewErr(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	payload := `{"token":25, "diffs":[{"ts":30052602, "diff":0.010173, "mode": -1, "color": 0},{"ts`

	parser := NewSurfaceFlingerParser()
	require.NotNil(parser)

	_, err := parser.Parse(payload)
	assert.NotNil(err)
}

func TestParseSFInvalid(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	payload := `should not parse this`

	parser := NewSurfaceFlingerParser()
	require.NotNil(parser)

	log, err := parser.Parse(payload)
	assert.Nil(err)
	assert.Nil(log)
}
