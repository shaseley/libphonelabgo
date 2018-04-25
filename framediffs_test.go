package libphonelabgo

import (
	phonelab "github.com/shaseley/phonelab-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

type counterHandler struct {
	expected int
	count    int
	t        *testing.T
}

func (h *counterHandler) Handle(log interface{}) interface{} {
	switch log.(type) {
	case *FrameDiffSample:
		h.count += 1
	}
	return nil
}

func (h *counterHandler) Finish() {
	assert.Equal(h.t, h.expected, h.count)
}

type counterGen struct {
	t *testing.T
}

func (g *counterGen) GenerateProcessor(source *phonelab.PipelineSourceInstance,
	kwargs map[string]interface{}) phonelab.Processor {

	expected := 0
	if v, ok := kwargs["expected"]; ok {
		expected = v.(int)
	}

	return phonelab.NewSimpleProcessor(source.Processor,
		&counterHandler{
			expected: expected,
			count:    0,
			t:        g.t,
		})
}

func TestFrameDiffEmitter(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	confString := `
source:
  type: files
  sources: ["./test/test.log"]
processors:
  - name: diffstream
    generator: diffs
    has_logstream: true
    parsers: ["SurfaceFlinger"]
    filters:
      - type: simple
        filter: "SurfaceFlinger"

  - name: main
    generator: counter
# Setting the next line to true would still work for this test,
# but it's not necessary -- we're just counting diffs.
    has_logstream: false
    inputs:
      - name: diffstream

sink:
  name: main
  args:
    expected: 740
`

	env := phonelab.NewEnvironment()
	env.Parsers["SurfaceFlinger"] = func() phonelab.Parser { return NewSurfaceFlingerParser() }
	env.Processors["diffs"] = &FrameDiffEmitterGenerator{}
	env.Processors["counter"] = &counterGen{t}

	conf, err := phonelab.RunnerConfFromString(confString)
	require.Nil(err)
	require.NotNil(conf)

	runner, err := conf.ToRunner(env)
	require.Nil(err)
	require.NotNil(runner)

	t.Log(runner.Source)

	// Counts are checked by the handler
	errs := runner.Run()
	assert.Equal(0, len(errs))
}

////////////////////////////////////////////////////////////////////////////////

type interlaceVerifyHandler struct {
	interval int64
	prevMs   int64
	t        *testing.T
}

func (h *interlaceVerifyHandler) Handle(log interface{}) interface{} {
	switch t := log.(type) {
	case *FrameDiffSample:
		if h.prevMs > 0 {
			assert.True(h.t, h.prevMs+2*h.interval >= t.Timestamp)
		}
		h.prevMs = t.Timestamp
	}
	return nil
}

func (h *interlaceVerifyHandler) Finish() {}

type interlaceVerifyGen struct {
	t *testing.T
}

func (g *interlaceVerifyGen) GenerateProcessor(source *phonelab.PipelineSourceInstance,
	kwargs map[string]interface{}) phonelab.Processor {

	interlace := 0
	if v, ok := kwargs["interlace"]; ok {
		interlace = v.(int)
	}

	return phonelab.NewSimpleProcessor(source.Processor,
		&interlaceVerifyHandler{
			interval: int64(interlace),
			t:        g.t,
		})
}

func TestFrameDiffInterlaceZeros(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	confString := `
source:
  type: files
  sources: ["./test/test.log"]

sink:
  name: main
  args: &args
    interlace: 40

processors:
  - name: diffstream
    generator: diffs
    has_logstream: true
    parsers: ["SurfaceFlinger"]
    filters:
      - type: simple
        filter: "SurfaceFlinger"

  - name: main
    generator: verifier
    has_logstream: false
    inputs:
      - name: diffstream
        args: *args
`

	env := phonelab.NewEnvironment()
	env.Parsers["SurfaceFlinger"] = func() phonelab.Parser { return NewSurfaceFlingerParser() }
	env.Processors["diffs"] = &FrameDiffEmitterGenerator{}
	env.Processors["verifier"] = &interlaceVerifyGen{t}

	conf, err := phonelab.RunnerConfFromString(confString)
	require.Nil(err)
	require.NotNil(conf)

	runner, err := conf.ToRunner(env)
	require.Nil(err)
	require.NotNil(runner)

	//	t.Log(runner.Source)

	// Counts are checked by the handler
	errs := runner.Run()
	assert.Equal(0, len(errs))
}
