package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGraphDefinition(t *testing.T) {
	var softnas SoftnasPlugin

	graphdef := softnas.GraphDefinition()
	if len(graphdef) != 4 {
		t.Errorf("GetTempfilename: %d should be 4", len(graphdef))
	}
	assert.EqualValues(t, "SoftNas StorageName", graphdef["softnas.storagename"].Label)
	assert.EqualValues(t, "SoftNas StorageData", graphdef["softnas.storagedata"].Label)
	assert.EqualValues(t, "SoftNas MemoryName", graphdef["softnas.memoryname"].Label)
	assert.EqualValues(t, "SoftNas MemoryData", graphdef["softnas.memorydata"].Label)
}

func TestbyteSizeConvert(t *testing.T) {
	stub := []string{"1,000K", "1,000M", "1,000G", "1,000T", "1,000"}
	for _, v := range stub {
		stat, err := byteSizeConvert(v)
		assert.Nil(t, err)
		if strings.HasSuffix(v, "K") {
			assert.EqualValues(t, 1.024e+06, stat)
		} else if strings.HasSuffix(v, "M") {
			assert.EqualValues(t, 1.048576e+09, stat)
		} else if strings.HasSuffix(v, "G") {
			assert.EqualValues(t, 1.073741824e+12, stat)
		} else if strings.HasSuffix(v, "T") {
			assert.EqualValues(t, 1.099511627776e+15, stat)
		} else {
			assert.EqualValues(t, 1000, stat)
		}
	}
}
