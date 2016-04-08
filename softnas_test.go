package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSoftnasSessionID(t *testing.T) {
	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, `{"success" : true, "session_id" : 12345, "result" : {}}`)
			}))
	defer ts.Close()
	id, err := getSoftnasSessionID("./softnas-cmd_test", ts.URL, "softnas", "Pass4W0rd")
	assert.Nil(t, err)
	assert.EqualValues(t, 12345, id)
}

func TestFetchMetrics(t *testing.T) {

	mux := http.NewServeMux()
	mux.HandleFunc(
		"/overview",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"success" : true, "session_id" : 12345, "result" : {"success":true,"msg":"","records":[{"storage_name":"44.8G Free\n(100.0%)","storage_data":99.99897820609},{"storage_name":"480.0K Used\n(0.0%)","storage_data":0.0010217939104395},{"memory_name":"666.7K\nCache Used\n(0.1%)","memory_data":0.064876091113447},{"memory_name":"1,002.9M\nCache Free\n(99.9%)","memory_data":99.935123908887}],"total":4}}`)
		},
	)
	mux.HandleFunc(
		"/perfmon",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"result": {"msg": "", "records":[{"arc_hitpercent": 0,"arc_hits": 0,"arc_miss": 0,"arc_read": 0,"arc_size": 0,"arc_target": 0,"cpu": 0,"io_diskreads": 0,"io_diskwrites": 0,"io_netreads": 0,"io_netwrites": 0,"iops_cifs": 0,"iops_iscsi": 0,"iops_nfs": 0,"latency_cifs": 0,"latency_iscsi": 0,"latency_nfs": 0,"time": "09:15"}],"success": true,"total": 1},"session_id": 12345,"success": true}`)
		},
	)
	ts := httptest.NewServer(mux)
	defer ts.Close()
	var softnas SoftnasPlugin
	softnas.Command = "./softnas-cmd_test"
	softnas.BaseURL = ts.URL
	softnas.SessionID = "12345"
	stat, err := softnas.FetchMetrics()
	assert.Nil(t, err)
	assert.EqualValues(t, 491520, stat["storagename_used"])
	assert.EqualValues(t, 4.81036337152e+10, stat["storagename_free"])
	assert.EqualValues(t, 0.0010217939104395, stat["storagedata_used"])
	assert.EqualValues(t, 99.99897820609, stat["storagedata_free"])
	assert.EqualValues(t, 682700.8, stat["memoryname_used"])
	assert.EqualValues(t, 1.0516168704e+09, stat["memoryname_free"])
	assert.EqualValues(t, 0.064876091113447, stat["memorydata_used"])
	assert.EqualValues(t, 99.935123908887, stat["memorydata_free"])
}

func TestByteConvert(t *testing.T) {
	stub := []string{"1,000K", "1,000M", "1,000G", "1,000T", "1,000"}
	for _, v := range stub {
		stat, err := getSizeConvert(v)
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
	if len(graphdef) != 5 {
		t.Errorf("GetTempfilename: %d should be 4", len(graphdef))
	}
	assert.EqualValues(t, "SoftNas StorageName", graphdef["softnas.storagename"].Label)
	assert.EqualValues(t, "SoftNas StorageData", graphdef["softnas.storagedata"].Label)
	assert.EqualValues(t, "SoftNas MemoryName", graphdef["softnas.memoryname"].Label)
	assert.EqualValues(t, "SoftNas MemoryData", graphdef["softnas.memorydata"].Label)
	assert.EqualValues(t, "SoftNas NumberOfArcCache", graphdef["softnas.numberofarccache"].Label)
}
