package libphonelabgo

import (
	phonelab "github.com/shaseley/phonelab-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

const testFile = "./test/test.log"

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

func TestParseSFFrameDiffWithGrid(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	payload := `{"token":9, "diffs":[{"ts":75496872, "diff":0.000000, "mode": 0, "color": 0, "wh": 8, "grid": []},{"ts":75496923, "diff":0.149197, "mode": 0, "color": 0, "wh": 8, "grid": [{"p":33, "v":0.004578},{"p":34, "v":0.049133}]}]}`

	parser := NewSurfaceFlingerParser()
	require.NotNil(parser)

	log, err := parser.Parse(payload)
	assert.Nil(err)
	assert.NotNil(log)
	diffLog, ok := log.(*SFFrameDiffLog)
	require.True(ok)

	expected := &SFFrameDiffLog{
		Token: 9,
		Diffs: []*SFFrameDiff{
			&SFFrameDiff{
				Timestamp:   75496872,
				PctDiff:     0.0,
				GridWH:      8,
				GridEntries: []*GridEntry{},
			},
			&SFFrameDiff{
				Timestamp: 75496923,
				PctDiff:   0.149197,
				GridWH:    8,
				GridEntries: []*GridEntry{
					&GridEntry{
						Position: 33,
						Value:    0.004578,
					},
					&GridEntry{
						Position: 34,
						Value:    0.049133,
					},
				},
			},
		},
	}

	require.True(reflect.DeepEqual(expected, diffLog))
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

type sfLogCounterGen struct {
	t *testing.T
}

func (g *sfLogCounterGen) GenerateProcessor(source *phonelab.PipelineSourceInstance,
	kwargs map[string]interface{}) phonelab.Processor {

	return phonelab.NewSimpleProcessor(source.Processor, &sfLogCounterHandler{
		t:         g.t,
		fpsCount:  0,
		diffCount: 0,
	})
}

type sfLogCounterHandler struct {
	fpsCount  int
	diffCount int
	t         *testing.T
}

func (handler *sfLogCounterHandler) Handle(log interface{}) interface{} {
	ll := log.(*phonelab.Logline)
	switch ll.Payload.(type) {
	case *SFFpsLog:
		handler.fpsCount += 1
	case *SFFrameDiffLog:
		handler.diffCount += 1
	}
	return nil
}

func (handler *sfLogCounterHandler) Finish() {
	assert.Equal(handler.t, 52, handler.fpsCount)
	assert.Equal(handler.t, 37, handler.diffCount)
}

func TestParseEndToEnd(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	confString := `
source:
  type: files
  sources: ["./test/test.log"]
processors:
  - name: main
    generator: counter
    has_logstream: true
    parsers: ["SurfaceFlinger"]
sink:
  name: main`

	env := phonelab.NewEnvironment()
	env.Parsers["SurfaceFlinger"] = func() phonelab.Parser { return NewSurfaceFlingerParser() }
	env.Processors["counter"] = &sfLogCounterGen{t}

	conf, err := phonelab.RunnerConfFromString(confString)
	require.Nil(err)
	require.NotNil(conf)

	runner, err := conf.ToRunner(env)
	require.Nil(err)
	require.NotNil(runner)

	t.Log(runner.Source)

	// Checks are done during the run
	errs := runner.Run()
	assert.Equal(0, len(errs))
}
