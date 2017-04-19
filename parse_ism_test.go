package libphonelabgo

import (
	phonelab "github.com/shaseley/phonelab-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIMSParser(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)
	require := require.New(t)

	parser := NewIMSLifeCycleParser()

	payload := `{"Action":"onStartInputView","Time":1492483700860,"UpTimeNs":158108566813,"UpTimeMs":158108,"timestamp":1492483700860,"uptimeNanos":158108619260,"LogFormat":"1.1"}`

	expected := &IMSLifeCycleLog{
		PLLog: phonelab.PLLog{
			LogFormat:   "1.1",
			UptimeNanos: 158108619260,
			Timestamp:   1492483700860,
		},
		UpTimeNs: 158108566813,
		Action:   "onStartInputView",
	}

	res, err := parser.Parse(payload)
	require.Nil(err)

	assert.Equal(expected, res)

	payload = `{"Action":"onFinishInputView","Time":1492483718447,"UpTimeNs":175695147952,"UpTimeMs":175695,"timestamp":1492483718447,"uptimeNanos":175695167900,"LogFormat":"1.1"}`

	expected = &IMSLifeCycleLog{
		PLLog: phonelab.PLLog{
			LogFormat:   "1.1",
			UptimeNanos: 175695167900,
			Timestamp:   1492483718447,
		},
		UpTimeNs: 175695147952,
		Action:   "onFinishInputView",
	}

	res, err = parser.Parse(payload)
	require.Nil(err)

	assert.Equal(expected, res)

}
