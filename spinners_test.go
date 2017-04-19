package libphonelabgo

import (
	phonelab "github.com/shaseley/phonelab-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"sync"
	"testing"
)

type SpinnerDataCollector struct {
	t     *testing.T
	total int
	sync.Mutex
}

func (dc *SpinnerDataCollector) OnData(data interface{}) {
	dc.Lock()
	defer dc.Unlock()

	if o, ok := data.(*SpinnerCollectorOutput); ok {
		dc.total += 1
		dc.t.Log(o.Json())
	}
}

func (dc *SpinnerDataCollector) Finish() {}

func TestSpinners(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	confString := `
data_collector: main
source:
  type: files
  sources: ["./test/test.log"]

processors:
  - name: diffstream
    generator: framediffs
    has_logstream: true
    parsers:
      - &SF SurfaceFlinger
    filters:
      - type: simple
        filter: *SF

  - name: detector 
    generator: spinners
    has_logstream: false
    inputs:
      - name: diffstream
        args:
          interlaceZeros: true

  - name: main
    generator: spinner_collector
    inputs:
      - name: detector
        args: *spinner_args

sink:
  name: main
  args: &spinner_args
    min: 0.001
    max: 4.000
    algo: voting
    votesIn: 7
    votesOut: 3
    ignoreZeros: true
`
	env := phonelab.NewEnvironment()
	AddParsers(env)
	AddProcessors(env)

	env.DataCollectors["main"] = func() phonelab.DataCollector {
		return &SpinnerDataCollector{
			t:     t,
			total: 0,
		}
	}

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

func TestSpinnerAlgoConf(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	confIn := map[string]interface{}{
		"min":         0.001,
		"max":         5.100,
		"algo":        "naive",
		"votesIn":     8,
		"votesOut":    2,
		"ignoreZeros": true,
		"group":       "group_name",
	}
	conf := NewSpinnerAlgoConf(confIn)
	require.NotNil(conf)

	assert.Equal(0.001, conf.Min)
	assert.Equal(5.1, conf.Max)
	assert.Equal("naive", conf.Name)
	assert.Equal(8, conf.NumVotesIn)
	assert.Equal(2, conf.NumVotesOut)
	assert.Equal(true, conf.IgnoreZeros)
	assert.Equal("group_name", conf.Group)
}

func TestSpinnerAlgoConfNils(t *testing.T) {
	t.Parallel()

	// This should not panic the system, but can if we don't handle nils
	// properly.
	defer func() {
		if r := recover(); r != nil {
			t.Log("The test panicked!")
			t.FailNow()
		}
	}()

	assert := assert.New(t)
	require := require.New(t)

	confIn := map[string]interface{}{
		"min":         nil,
		"max":         nil,
		"algo":        nil,
		"votesIn":     nil,
		"votesOut":    nil,
		"ignoreZeros": nil,
		"group":       nil,
	}

	conf := NewSpinnerAlgoConf(confIn)
	require.NotNil(conf)

	assert.True(reflect.DeepEqual(&SpinnerAlgoConf{}, conf))
}

func TestSpinnerAlgoConfIntMinMax(t *testing.T) {
	t.Parallel()

	// This should not panic the system, but can if we don't handle nils
	// properly.
	defer func() {
		if r := recover(); r != nil {
			t.Log("The test panicked!")
			t.FailNow()
		}
	}()

	assert := assert.New(t)
	require := require.New(t)

	confIn := map[string]interface{}{
		"min":         1,
		"max":         4,
		"algo":        nil,
		"votesIn":     nil,
		"votesOut":    nil,
		"ignoreZeros": nil,
		"group":       nil,
	}

	conf := NewSpinnerAlgoConf(confIn)
	require.NotNil(conf)

	expected := &SpinnerAlgoConf{
		Min: 1.0,
		Max: 4.0,
	}

	assert.True(reflect.DeepEqual(expected, conf))
}

func TestSpinnerYamlInheritence(t *testing.T) {
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
    parsers:
      - &SF SurfaceFlinger
    filters:
      - type: simple
        filter: *SF

  - name: detector 
    generator: spinners
    has_logstream: false
    inputs:
      - name: diffstream
        args: {interlaceZeros: true}

  - name: main
    generator: spinner_collector
    inputs:
      - name: detector
        args: *sargs

sink:
  name: main
  args: &sargs
    min: 0.001
    max: 4.000
    algo: voting
    votesIn: 7
    votesOut: 3
    ignoreZeros: true
`
	conf, err := phonelab.RunnerConfFromString(confString)
	require.Nil(err)
	require.NotNil(conf)

	require.Equal(3, len(conf.Processors))
	proc := conf.Processors[2]
	require.Equal("main", proc.Name)
	require.Equal(1, len(proc.Inputs))

	t.Log(proc.Inputs[0].Args)
	t.Log(conf.Sink.Args)
	assert.True(reflect.DeepEqual(proc.Inputs[0].Args,
		conf.Sink.Args))
}
