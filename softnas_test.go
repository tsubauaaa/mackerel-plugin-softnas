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
	id, err := fetchSessionID("./softnas-cmd_test", ts.URL, "softnas", "Pass4W0rd")
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
			fmt.Fprint(w, `{"result": {"msg": "", "records":[{"arc_hitpercent": 0,"arc_hits": 10,"arc_miss": 9,"arc_read": 8,"arc_size": 0,"arc_target": 0,"cpu": 0,"io_diskreads": 0,"io_diskwrites": 0,"io_netreads": 0,"io_netwrites": 0,"iops_cifs": 0,"iops_iscsi": 0,"iops_nfs": 0,"latency_cifs": 0,"latency_iscsi": 0,"latency_nfs": 0,"time": "09:15"}],"success": true,"total": 1},"session_id": 12345,"success": true}`)
		},
	)
	mux.HandleFunc(
		"/pooldetails",
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"success" : true, "session_id" : 31214, "result" : {"success":true,"msg":"Pool details request for pool '' was successful.","records":[{"name":"pool1","status":"ONLINE","read_errors":"0","write_errors":"0","checksum_errors":"0","read_IOPS":"10","write_IOPS":"11","read_bandwidth":"0","write_bandwidth":"0","extended":"","scrub":"none requested"},{"name":"&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;\/dev\/s3-0","status":"ONLINE","read_errors":"0","write_errors":"0","checksum_errors":"0","read_IOPS":"9","write_IOPS":"8","read_bandwidth":"0","write_bandwidth":"0","extended":"","scrub":"none requested"}],"total":2}}`)
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
	assert.EqualValues(t, 10, stat["arc_hits"])
	assert.EqualValues(t, 9, stat["arc_miss"])
	assert.EqualValues(t, 8, stat["arc_read"])
	assert.EqualValues(t, 10, stat["read_iops"])
	assert.EqualValues(t, 11, stat["write_iops"])
}

func TestByteConvert(t *testing.T) {
	stub := []string{"1,000K", "1,000M", "1,000G", "1,000T", "1,000"}
	for _, v := range stub {
		stat, err := convertUnit(v)
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

func TestMetricsAgerage(t *testing.T) {
	stub := []float64{1.0, 2.0, 3.0, 4.0, 5.0, 6.0}
	stub0 := []float64{0.0, 0.0, 0.0, 0.0, 0.0, 0.0}
	stat := culculateAverage(stub)
	assert.EqualValues(t, 3.5, stat)
	stat = culculateAverage(stub0)
	assert.EqualValues(t, 0.0, stat)
}

func TestGraphDefinition(t *testing.T) {
	var softnas SoftnasPlugin

	graphdef := softnas.GraphDefinition()
	if len(graphdef) != 6 {
		t.Errorf("GetTempfilename: %d should be 6", len(graphdef))
	}
	assert.EqualValues(t, "SoftNas Storage Size", graphdef["softnas.storagename"].Label)
	assert.EqualValues(t, "SoftNas Storage Usage", graphdef["softnas.storagedata"].Label)
	assert.EqualValues(t, "SoftNas Cache Memory Size", graphdef["softnas.memoryname"].Label)
	assert.EqualValues(t, "SoftNas Cache Memory Usage", graphdef["softnas.memorydata"].Label)
	assert.EqualValues(t, "SoftNas ARC Cache", graphdef["softnas.numberofarccache"].Label)
	assert.EqualValues(t, "SoftNas Read/Write Pool IOPS", graphdef["softnas.pooliops"].Label)
}
