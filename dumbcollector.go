package libphonelabgo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
)

type DumbCollector struct {
	Data            []interface{}
	CheckFunc       func(interface{}) bool
	PersistOnFinish bool
	Filename        string
	sync.Mutex
}

func NewDumbCollector() *DumbCollector {
	return &DumbCollector{
		Data: make([]interface{}, 0),
	}
}

func (dc *DumbCollector) OnData(data interface{}) {
	dc.Lock()
	defer dc.Unlock()

	if dc.CheckFunc == nil || dc.CheckFunc(data) {
		dc.Data = append(dc.Data, data)
	}
}

func (dc *DumbCollector) Finish() {
	if dc.PersistOnFinish {
		if err := dc.DumpJson(dc.Filename); err != nil {
			fmt.Fprintf(os.Stderr, "Error persisting data: %v\n", err)
		}
	}
}

func (dc *DumbCollector) DumpJson(outFile string) error {
	outputBytes, err := json.MarshalIndent(dc.Data, "", "\t")

	if err != nil {
		return fmt.Errorf("Error marshalling data: %v", err)
	} else if len(outFile) == 0 {
		fmt.Println(string(outputBytes))
	} else {
		return ioutil.WriteFile(outFile, outputBytes, 0644)
	}
	return nil
}
