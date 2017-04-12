package libphonelabgo

import (
	"fmt"
	phonelab "github.com/shaseley/phonelab-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

type inputTester struct {
	skipScrolls      bool
	skipKeys         bool
	expectedGestures []int
	actualGestures   []int
	expectedKeys     []int
	actualKeys       []int
	t                *testing.T
}

func (tester *inputTester) Handle(log interface{}) interface{} {
	if event, ok := log.(*TouchScreenEvent); ok && event != nil {
		add := true

		if event.What == TouchScreenEventKey {
			add = !tester.skipKeys
		} else if event.What == TouchScreenEventScroll {
			add = !tester.skipScrolls
		}

		if add {
			tester.actualGestures = append(tester.actualGestures, event.What)
			if event.What == TouchScreenEventKey {
				tester.actualKeys = append(tester.actualKeys, event.Code)
			}
		}
	}
	return nil
}

func (tester *inputTester) Finish() {
	assert.Equal(tester.t, tester.expectedGestures, tester.actualGestures)
	assert.Equal(tester.t, tester.expectedKeys, tester.actualKeys)
}

type inputTesterGenerator struct {
	skipScrolls      bool
	skipKeys         bool
	expectedGestures []int
	expectedKeys     []int
	t                *testing.T
}

func (itg *inputTesterGenerator) GenerateProcessor(source *phonelab.PipelineSourceInstance,
	kwargs map[string]interface{}) phonelab.Processor {

	return phonelab.NewSimpleProcessor(source.Processor, &inputTester{
		skipScrolls:      itg.skipScrolls,
		skipKeys:         itg.skipKeys,
		expectedGestures: itg.expectedGestures,
		actualGestures:   make([]int, 0),
		expectedKeys:     itg.expectedKeys,
		actualKeys:       make([]int, 0),
		t:                itg.t,
	})
}

func testInputProcCommon(t *testing.T, expectedGestures, expectedKeys []int, skipScrolls, skipKeys bool, file string) {
	confString := fmt.Sprintf(`
source:
  type: files
  sources: ["%v"]

processors:
  - name: input
    has_logstream: true
    parsers:
      - InputDispatcher-MotionEvent
      - InputDispatcher-KeyEvent
    filters:
      - type: simple
        filter: InputDispatcher

  - name: main
    generator: tester
    inputs:
      - name: input

sink:
  name: main
`, file)

	assert := assert.New(t)
	require := require.New(t)

	env := phonelab.NewEnvironment()
	RegisterInputFlingerParsers(env)

	env.Processors["input"] = &InputProcessorGenerator{}
	env.Processors["tester"] = &inputTesterGenerator{
		skipScrolls:      skipScrolls,
		skipKeys:         skipKeys,
		expectedGestures: expectedGestures,
		expectedKeys:     expectedKeys,
		t:                t,
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
	if len(errs) > 0 {
		t.Log(errs)
	}
	assert.Equal(0, len(errs))
}

func TestInputProcessorSingleTap(t *testing.T) {
	t.Parallel()

	expected := []int{
		TouchScreenEventTap,
	}

	testInputProcCommon(t, expected, []int{}, true, true, "test/input/tap.log")
}

func TestInputProcessorTripleTap(t *testing.T) {
	t.Parallel()

	expected := []int{
		TouchScreenEventTap,
		TouchScreenEventTap,
		TouchScreenEventTap,
	}

	testInputProcCommon(t, expected, []int{}, true, true, "test/input/tripletap.log")
}

func TestInputProcessorTapTapScrollTap(t *testing.T) {
	t.Parallel()

	expected := []int{
		TouchScreenEventTap,
		TouchScreenEventTap,
		TouchScreenEventScrollStart,
		TouchScreenEventScrollEnd,
		TouchScreenEventTap,
	}

	testInputProcCommon(t, expected, []int{}, true, true, "test/input/taptapscrolltap.log")
}

func TestInputProcessorComplex(t *testing.T) {
	t.Parallel()

	expected := []int{
		TouchScreenEventTap,
		TouchScreenEventTap,
		TouchScreenEventScrollStart,
		TouchScreenEventScrollEnd,
		TouchScreenEventTap,
		TouchScreenEventScrollStart,
		TouchScreenEventScrollEnd,
		TouchScreenEventScrollStart,
		TouchScreenEventScrollEnd,
		TouchScreenEventTap,
		TouchScreenEventTap,
	}

	testInputProcCommon(t, expected, []int{}, true, true, "test/input/complex.log")
}

func TestInputProcessorHardKeys(t *testing.T) {
	t.Parallel()

	keys := []int{
		KEYCODE_VOLUME_UP,
		KEYCODE_VOLUME_DOWN,
		KEYCODE_POWER,
	}

	gestures := []int{
		TouchScreenEventKey,
		TouchScreenEventKey,
		TouchScreenEventKey,
	}

	testInputProcCommon(t, gestures, keys, false, false, "test/input/hardkeys.log")
}

func TestInputProcessorSoftKeys(t *testing.T) {
	t.Parallel()

	keys := []int{
		KEYCODE_BACK,
		KEYCODE_HOME,
		KEYCODE_BACK,
		KEYCODE_HOME,
	}

	gestures := []int{
		TouchScreenEventTap,
		TouchScreenEventKey,
		TouchScreenEventTap,
		TouchScreenEventKey,
		TouchScreenEventTap,
		TouchScreenEventKey,
		TouchScreenEventTap,
		TouchScreenEventKey,
	}

	testInputProcCommon(t, gestures, keys, false, false, "test/input/softkeys.log")
}
