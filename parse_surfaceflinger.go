package libphonelabgo

import (
	"encoding/json"
	"fmt"
	phonelab "github.com/shaseley/phonelab-go"
	"strings"
)

const (
	nexus6ScreenW = 1440
	nexus6ScreenH = 2560
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
	Timestamp   int64        `json:"ts"`
	PctDiff     float64      `json:"diff"`
	Mode        int          `json:"mode"`
	HasColor    int          `json:"color"`
	GridWH      int          `json:"wh"`
	GridEntries []*GridEntry `json:"grid"`
	Grid        *ScreenGrid  `json:"-"`
}

type GridEntry struct {
	Position int     `json:"p"`
	Value    float64 `json:"v"`
}

type ScreenGrid struct {
	grid  [][]float64
	props *screenGridProps
}

type screenGridProps struct {
	rows        int
	cols        int
	gridWH      int
	screenW     int
	screenH     int
	edgeMultRow float64
	edgeMultCol float64
	pixelsPerWH float64
}

var allScreenGrids []*screenGridProps

func init() {
	allScreenGrids = []*screenGridProps{
		&screenGridProps{
			screenW:     1440,
			screenH:     2560,
			gridWH:      8,
			rows:        8,
			cols:        5,
			edgeMultRow: 1.0,
			edgeMultCol: 2.0,
			pixelsPerWH: 2560 / 8,
		},
	}
}

// Returns -1 if out of expected bounds
func (props *screenGridProps) entryPosToGridPos(pos int) (row, col int) {
	row = pos / props.gridWH
	col = pos - (row * props.gridWH)

	if row >= props.rows || col >= props.cols {
		row, col = -1, -1
		return
	}

	// SF grid starts in lower left, but input coordinates start in upper left,
	// so we'll mirror the grid height pos.
	row = (props.gridWH - 1) - row

	return row, col
}

func (props *screenGridProps) gridPosFromXY(x, y float64) (row, col int) {

	row = int(y / props.pixelsPerWH)
	col = int(x / props.pixelsPerWH)

	if col < 0 || col >= props.cols ||
		row < 0 || row >= props.rows {
		row, col = -1, -1
	}

	return
}

func (diff *SFFrameDiff) initScreenGrid(props *screenGridProps) {

	diff.Grid = &ScreenGrid{
		props: props,
	}

	grid := make([][]float64, props.rows, props.rows)

	for i := 0; i < props.rows; i += 1 {
		grid[i] = make([]float64, props.cols, props.cols)
	}

	// Old format logs didn't have gridded diffs
	if props.rows == 1 && props.cols == 1 {
		grid[0][0] = diff.PctDiff
		return
	}

	for _, entry := range diff.GridEntries {
		if row, col := props.entryPosToGridPos(entry.Position); row < 0 || col < 0 {
			panic("New grid position < 0!")
		} else {
			grid[row][col] = entry.Value
		}
	}

	diff.Grid.grid = grid
}

type PixelConnectivity int

const (
	OneConnected   PixelConnectivity = 0
	FourConnected                    = 4
	EightConnected                   = 8
)

type position struct {
	row int
	col int
}

func (diff *SFFrameDiff) LocalDiff(connectivity PixelConnectivity, x, y float64) (float64, int, error) {
	if connectivity != FourConnected && connectivity != EightConnected && connectivity != OneConnected {
		return 0.0, 0, fmt.Errorf("Invalid connectivity '%v', expected 4 or 8", connectivity)
	}

	props := diff.Grid.props

	row, col := props.gridPosFromXY(x, y)
	if row < 0 || col < 0 {
		return 0.0, 0, fmt.Errorf("Invalid positions: x=%v, y=%v is out of bounds", x, y)
	}

	positions := make([]*position, 0, int(connectivity)+1)

	positions = append(positions, &position{row, col})

	if connectivity != OneConnected {
		positions = append(positions, &position{row - 1, col})
		positions = append(positions, &position{row + 1, col})
		positions = append(positions, &position{row, col - 1})
		positions = append(positions, &position{row, col + 1})
	}

	if connectivity == EightConnected {
		positions = append(positions, &position{row - 1, col - 1})
		positions = append(positions, &position{row - 1, col + 1})
		positions = append(positions, &position{row + 1, col + 1})
		positions = append(positions, &position{row + 1, col - 1})
	}

	sum := float64(0)
	count := 0

	for _, p := range positions {
		if p.row >= 0 && p.col >= 0 && p.row < props.rows && p.col < props.cols {
			pctDiff := diff.Grid.grid[p.row][p.col]
			if p.row == props.rows-1 {
				pctDiff *= props.edgeMultRow
			}
			if p.col == props.cols-1 {
				pctDiff *= props.edgeMultCol
			}
			sum += pctDiff
			count += 1
		}
	}

	return sum / float64(count), count, nil
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
