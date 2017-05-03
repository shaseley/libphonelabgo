package libphonelabgo

import (
	phonelab "github.com/shaseley/phonelab-go"
)

// config.go has global configuration and setup functions.

// Global library configuration options. These are meant to be specified by
var GlobalConf = struct {
	// Whether or not processors should use the Android sysTime() timestamp if
	// available. In some cases, this eliminates the need for using the extra
	// timesync processor, like when the only required timestamps both have this
	// available.
	UseSysTime bool
}{
	false,
}

func AddParsers(env *phonelab.Environment) {
	// SurfaceFlinger
	env.RegisterParserGenerator("SurfaceFlinger", NewSurfaceFlingerParser)

	// InputFlinger
	env.RegisterParserGenerator("InputDispatcher-MotionEvent", NewIFMotionEventParser)
	env.RegisterParserGenerator("InputDispatcher-KeyEvent", NewIFKeyEventParser)

	// InputServiceManager
	env.RegisterParserGenerator("InputMethodService-LifeCycle-QoE", NewIMSLifeCycleParser)
}

// Add all known processors to the enviroment. Any arguments needed for the
// processors are configured through yaml args.
func AddProcessors(env *phonelab.Environment) {
	// Frame diffs
	env.Processors["framediffs"] = &FrameDiffEmitterGenerator{}

	// Spinners
	env.Processors["spinners"] = &SpinnerAlgoGenerator{}
	env.Processors["spinner_stitcher"] = &SpinnerStitcherGen{}
	env.Processors["spinner_collector"] = &SpinnerCollectorGenerator{}

	// Input
	env.Processors["input_gestures"] = &InputProcessorGenerator{}

	// Input + Diffs
	env.Processors["input_diffs"] = &InputDiffProcessorGenerator{}

	// Time sync (preprocessor)
	env.Processors["timesync"] = &TimeSyncPreprocessorGenerator{}

	// Frame samples
	env.Processors["frametimes"] = &FrameRefreshEmitterGen{}
}
