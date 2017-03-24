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
	h.count += 1
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

func TestFrameDiffOffset(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	tests := []struct {
		ts       int64
		offset   int64
		expected float64
	}{
		{123645302254, 0, 123.645302254},
		{123645302254, 70000000, 123.715302254},
	}

	for _, test := range tests {
		assert.Equal(test.expected, adjustTimestampNsToS(test.ts, test.offset))
	}
}
