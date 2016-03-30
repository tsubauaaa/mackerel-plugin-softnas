package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testSessionIDHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"success" : true, "session_id" : 12345, "result" : {}}`)
	return
})

var testOverviewHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"success" : true, "session_id" : 12345, "result" : {"success":true,"msg":"","records":[{"storage_name":"44.8G Free\n(100.0%)","storage_data":99.99897820609},{"storage_name":"480.0K Used\n(0.0%)","storage_data":0.0010217939104395},{"memory_name":"666.7K\nCache Used\n(0.1%)","memory_data":0.064876091113447},{"memory_name":"1,002.9M\nCache Free\n(99.9%)","memory_data":99.935123908887}],"total":4}}`)
	return
})

func TestSessionID(t *testing.T) {
	http.HandleFunc("/sessionid", testSessionIDHandler)
	http.HandleFunc("/overview", testOverviewHandler)
	http.ListenAndServe(":8080", nil)
	defer.close
	id, err := getSoftnasSessionID("./softnas-cmd_test", "softnas", "Pass4W0rd")
	assert.Nil(t, err)
	assert.EqualValues(t, 12345, id)
}

func TestConvert(t *testing.T) {
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
