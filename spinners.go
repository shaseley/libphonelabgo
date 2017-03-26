package libphonelabgo

import (
	"encoding/json"
	phonelab "github.com/shaseley/phonelab-go"
)

type Spinner struct {
	StartTimeMs int64 `json:"start_time_ms"`
	EndTimeMs   int64 `json:"end_time_ms"`
	DurationMs  int64 `json:"duration_ms"`

	TraceTimeStart float64 `json:"trace_time_start"`
	TraceTimeEnd   float64 `json:"trace_time_end"`
}

func (s *Spinner) MonotonicTimestamp() float64 {
	return s.TraceTimeStart
}

type SpinnerAlgoGenerator struct{}

// All spinner algorithm parameters. Not all algorithms use the same parameters,
// but we put them all in one struct to make things simpler.
type SpinnerAlgoConf struct {
	Name        string
	Min         float64
	Max         float64
	IgnoreZeros bool
	NumVotesIn  int
	NumVotesOut int
}

func NewSpinnerAlgoConf(kwargs map[string]interface{}) *SpinnerAlgoConf {
	p := &SpinnerAlgoConf{}

	if v, ok := kwargs["min"]; ok {
		p.Min = v.(float64)
	}
	if v, ok := kwargs["max"]; ok {
		p.Max = v.(float64)
	}
	if v, ok := kwargs["ignoreZeros"]; ok {
		p.IgnoreZeros = v.(bool)
	}
	if v, ok := kwargs["algo"]; ok {
		p.Name = v.(string)
	}
	if v, ok := kwargs["votesIn"]; ok {
		p.NumVotesIn = v.(int)
	}
	if v, ok := kwargs["votesOut"]; ok {
		p.NumVotesOut = v.(int)
	}

	return p
}

func (g *SpinnerAlgoGenerator) GenerateProcessor(source *phonelab.PipelineSourceInstance,
	kwargs map[string]interface{}) phonelab.Processor {

	conf := NewSpinnerAlgoConf(kwargs)

	switch conf.Name {
	case "naive":
		return phonelab.NewSimpleProcessor(source.Processor,
			NewNaiveSpinnerAlgo(conf))
	case "voting":
		return phonelab.NewSimpleProcessor(source.Processor,
			NewVotingSpinnerAlgo(conf))
	default:
		// TODO: this should be able to return an error
		panic("Cannot find algo '" + conf.Name + "'!")
	}
}

// TODO:
//	Build processors to run the spinner algorithms
//		-> Output spinners, one at a time
//		-> Parameterize them so we can specify from yaml
//
//	Try to extract out common logic

////////////////////////////////////////////////////////////////////////////////
// Common spinner state
type spinnerState struct {
	isSpinner  bool
	startMs    int64
	startTrace float64
}

func (state *spinnerState) setState(isSpinner bool) {
	state.isSpinner = isSpinner
}

func (state *spinnerState) markStartTime(sample *FrameDiffSample) {
	state.startMs = sample.Timestamp
	state.startTrace = sample.MonotonicTimestamp()
}

func (state *spinnerState) endSpinner(sample *FrameDiffSample) *Spinner {
	return &Spinner{
		StartTimeMs:    state.startMs,
		EndTimeMs:      sample.Timestamp,
		DurationMs:     sample.Timestamp - state.startMs,
		TraceTimeStart: state.startTrace,
		TraceTimeEnd:   sample.MonotonicTimestamp(),
	}
}

////////////////////////////////////////////////////////////////////////////////
// NaiveSpinnerAlgo is just that, very naive. It classifies on a per-frame basis
// without caring about previous or future frames.
type NaiveSpinnerAlgo struct {
	Conf  SpinnerAlgoConf
	state *spinnerState
}

func NewNaiveSpinnerAlgo(conf *SpinnerAlgoConf) *NaiveSpinnerAlgo {
	return &NaiveSpinnerAlgo{
		Conf:  *conf,
		state: &spinnerState{},
	}
}

func (algo *NaiveSpinnerAlgo) Handle(log interface{}) interface{} {
	// We're expecting only frame diff samples
	sample, ok := log.(*FrameDiffSample)
	if !ok {
		return nil
	}

	diff := sample.PctDiff

	// Only if we have a state change do we need to do anything,
	// and if 0.0 and ignore zeros, we don't change states.
	if diff > float64(0.0) || !algo.Conf.IgnoreZeros {
		isSpinner := (diff > algo.Conf.Min && diff < algo.Conf.Max)

		if isSpinner && !algo.state.isSpinner {
			algo.state.setState(true)
			algo.state.markStartTime(sample)
		} else if !isSpinner && algo.state.isSpinner {
			algo.state.setState(false)
			return algo.state.endSpinner(sample)
		}
	}
	return nil
}

func (algo *NaiveSpinnerAlgo) Finish() {}

////////////////////////////////////////////////////////////////////////////////
// Simple voting classifier

type VotingSpinnerAlgo struct {
	Conf SpinnerAlgoConf
	// State
	state       *spinnerState
	votesNeeded int
}

func NewVotingSpinnerAlgo(conf *SpinnerAlgoConf) *VotingSpinnerAlgo {
	c := &VotingSpinnerAlgo{
		Conf:        *conf,
		state:       &spinnerState{},
		votesNeeded: conf.NumVotesIn,
	}
	return c
}

func (algo *VotingSpinnerAlgo) voteFor(sample *FrameDiffSample) {
	if algo.state.isSpinner {
		// positive reinforcement, reset votes
		algo.votesNeeded = algo.Conf.NumVotesOut
	} else {
		// A step in the right direction. But, how many steps?

		if algo.votesNeeded == algo.Conf.NumVotesIn {
			// No samples yet, mark the start time
			algo.state.markStartTime(sample)
		}

		algo.votesNeeded -= 1

		if algo.votesNeeded <= 0 {
			// State change, we've identified a spinner
			algo.state.setState(true)
			algo.votesNeeded = algo.Conf.NumVotesOut
		}
	}
}

func (algo *VotingSpinnerAlgo) voteAgainst(sample *FrameDiffSample) *Spinner {
	if !algo.state.isSpinner {
		// negative reinforcement, reset votes
		algo.votesNeeded = algo.Conf.NumVotesIn
	} else {
		algo.votesNeeded -= 1
		if algo.votesNeeded <= 0 {
			// State change, the spinner is done
			algo.state.setState(false)
			algo.votesNeeded = algo.Conf.NumVotesIn

			// Return the previous spinner
			return algo.state.endSpinner(sample)
		}
	}
	return nil
}

func (algo *VotingSpinnerAlgo) Handle(log interface{}) interface{} {
	// 	Simple state machine:
	// 		Require N consecutive samples with the bounds to classify as yes,
	//  	and require M consecutive samples outside of the bounds to classify
	//  	as no.

	// We're expecting only frame diff samples
	sample, ok := log.(*FrameDiffSample)
	if !ok {
		return nil
	}

	diff := sample.PctDiff
	cur := (diff > algo.Conf.Min && diff < algo.Conf.Max)

	if cur {
		algo.voteFor(sample)
		return nil
	} else {
		// Only votes against can return a spinner.
		if res := algo.voteAgainst(sample); res != nil {
			return res
		}
	}

	return nil
}

func (algo *VotingSpinnerAlgo) Finish() {}

////////////////////////////////////////////////////////////////////////////////
// Spinner Collector

type SpinnerCollectorProcessor struct {
	FileName    string
	SpinnerConf *SpinnerAlgoConf
	Source      phonelab.Processor
}

func NewSpinnerCollectorProcessor(inst *phonelab.PipelineSourceInstance, args map[string]interface{}) *SpinnerCollectorProcessor {

	res := &SpinnerCollectorProcessor{
		SpinnerConf: NewSpinnerAlgoConf(args),
		Source:      inst.Processor,
	}
	if v, ok := inst.Info["file_name"]; ok {
		res.FileName = v.(string)
	}
	return res
}

type SpinnerCollectorOutput struct {
	File     string           `json:"file"`
	Conf     *SpinnerAlgoConf `json:"conf"`
	Spinners []*Spinner       `json:"spinners"`
}

func (o *SpinnerCollectorOutput) Json() string {
	outputBytes, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return ""
	}
	return string(outputBytes)
}

func (p *SpinnerCollectorProcessor) Process() <-chan interface{} {
	outChan := make(chan interface{})

	go func() {
		inChan := p.Source.Process()
		spinners := make([]*Spinner, 0)

		for log := range inChan {
			if spinner, ok := log.(*Spinner); ok && spinner != nil {
				spinners = append(spinners, spinner)
			}
		}

		outChan <- &SpinnerCollectorOutput{
			File:     p.FileName,
			Conf:     p.SpinnerConf,
			Spinners: spinners,
		}
		close(outChan)
	}()

	return outChan
}

type SpinnerCollectorGenerator struct{}

func (g *SpinnerCollectorGenerator) GenerateProcessor(source *phonelab.PipelineSourceInstance,
	kwargs map[string]interface{}) phonelab.Processor {
	return NewSpinnerCollectorProcessor(source, kwargs)
}
