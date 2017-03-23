package libphonelabgo

import (
	"encoding/json"
	"fmt"
	phonelab "github.com/shaseley/phonelab-go"
	"strings"
)

// Example:
// 3a45bd43-82d2-4650-8571-039f48c0fdca 2016-12-02 03:03:28.15999782 453831 [184538.859034]   291   291 I SurfaceFlinger: {"fps":37.8, "tot_frames":84729, "prev_frames":84691, "cur_time": 184538788137586, "prev_time": 184537784018732}
type SFFpsLog struct {
	FPS           float64 `json:"fps"`
	TotalFrames   int     `json:"tot_frames"`
	PrevFrames    int     `json:"prev_frames"`
	SysTimestamp  int64   `json:"cur_time"`
	PrevTimestamp int64   `json:"prev_time"`
}

type SFFpsParserProps struct{}

func (p *SFFpsParserProps) New() interface{} {
	return &SFFpsLog{}
}

func NewSFFpsParer() phonelab.Parser {
	return phonelab.NewJSONParser(&SFFpsParserProps{})
}

// Example -- old logs -- malformed json :-(
//3a45bd43-82d2-4650-8571-039f48c0fdca 2016-12-02 03:03:27.885999782 453830 [184538.724191]   291   291 I SurfaceFlinger: {"token":1699, "diffs":[[184533085, 0.000000],[184533135, 0.000000],[184533880, 0.000000],[184533941, 0.493028],[184533994, 69.184029],[184534044, 100.000000],[184534095, 100.000000],[184534146, 99.305557],[184534196, 97.222221],[184534246, 93.315971],[184534297, 71.571182],[184534347, 0.000000],[184538215, 100.000000],[184538279, 0.010851],[184538333, 0.008138],[184538384, 0.007968],[184538429, 0.007629],[184538497, 0.008138],[184538563, 0.008816],[184538632, 0.008816]]}
type SFFrameDiffLog struct {
	Token int64          `json:"token"`
	Diffs []*SFFrameDiff `json:"diffs"`
}

type SFFrameDiff struct {
	Timestamp int64   `json:"ts"`
	PctDiff   float64 `json:"diff"`
	Mode      int     `json:"mode"`
	HasColor  int     `json:"color"`
}

type SFFrameDiffsJsonParserProps struct{}

func (p *SFFrameDiffsJsonParserProps) New() interface{} {
	return &SFFrameDiffLog{}
}

func NewSFFrameDiffsJsonParser() phonelab.Parser {
	return phonelab.NewJSONParser(&SFFrameDiffsJsonParserProps{})
}

////////////////////////////////////////////////////////////////////////////////

// SurfaceFlingerParser parses logs with the SurfaceFlinger tag.
// Currently, it handles FPS and frame diff logs.
type SurfaceFlingerParser struct {
	fpsJsonParser  phonelab.Parser
	diffJsonParser phonelab.Parser
}

// Create a new SurfaceFlingerParser
func NewSurfaceFlingerParser() phonelab.Parser {
	return &SurfaceFlingerParser{
		fpsJsonParser:  NewSFFpsParer(),
		diffJsonParser: NewSFFrameDiffsJsonParser(),
	}
}

// Parse payloads of logs with the SurfaceFlinger tag
func (parser *SurfaceFlingerParser) Parse(payload string) (interface{}, error) {
	if strings.Contains(payload, `{"fps":`) {
		// FPS log
		return parser.fpsJsonParser.Parse(payload)
	} else if strings.Contains(payload, `[{"ts":`) {
		// New, valid JSON log
		return parser.diffJsonParser.Parse(payload)
	} else if strings.Contains(payload, `"diffs":[[`) {
		// Old JSON with heterogeneous arrays
		return parseAndConvertOldDiffLog(payload)
	} else {
		// We can't parse it
		return nil, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Hacky, broken JSON frame diff log parsing

type oldSFDiffLog struct {
	Token int64             `json:"token"`
	Diffs []*oldSFFrameDiff `json:"diffs"`
}

type oldSFFrameDiff struct {
	Timestamp int64
	PctDiff   float64
}

func (l *oldSFFrameDiff) UnmarshalJSON(buf []byte) error {
	// We used to parse this manually, but there is a nice hack to get this into
	// a []interface{}: http://eagain.net/articles/go-json-array-to-struct/

	tmp := []interface{}{&l.Timestamp, &l.PctDiff}
	wantLen := len(tmp)

	if err := json.Unmarshal(buf, &tmp); err != nil {
		return err
	}

	if actual := len(tmp); actual != wantLen {
		return fmt.Errorf("wrong number of fields in Notification: %d != %d", actual, wantLen)
	}

	return nil
}

func parseAndConvertOldDiffLog(payload string) (interface{}, error) {
	var log *oldSFDiffLog

	if err := json.Unmarshal([]byte(payload), &log); err != nil {
		return nil, err
	}

	// Now, convert to the new format
	newLog := &SFFrameDiffLog{
		Token: log.Token,
		Diffs: make([]*SFFrameDiff, 0, len(log.Diffs)),
	}

	for _, diff := range log.Diffs {
		newLog.Diffs = append(newLog.Diffs,
			&SFFrameDiff{
				Timestamp: diff.Timestamp,
				PctDiff:   diff.PctDiff,
				Mode:      0,
				HasColor:  0,
			})
	}

	return newLog, nil
}
