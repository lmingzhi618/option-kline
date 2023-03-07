package regulator

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
)

func TestKline(t *testing.T) {
	ret := make([]float64, 0)
	for i := 0; i < 1000; i++ {
		ret = append(ret, RegulatorMap["GT"].AdjustPrice(&AdJustRequest{
			Price:     100.00,
			Precision: 4,
		}))
	}

	buf := bytes.NewBuffer(nil)
	for i := 0; i < 1000; i++ {
		buf.WriteString(fmt.Sprintf("%v, ", ret[i]))
	}

	ioutil.WriteFile("123.csv", buf.Bytes(), 0644)
}
