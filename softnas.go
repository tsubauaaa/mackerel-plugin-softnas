package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	mp "github.com/mackerelio/go-mackerel-plugin-helper"
)

// SoftnasPlugin mackerel plugin for softnas
type SoftnasPlugin struct {
	Command   string
	BaseURL   string
	User      string
	Password  string
	SessionID string
	PoolName  []string
}

// LoginResult softnas-cmd login result for SessionID
type LoginResult struct {
	Success   bool `json:"success"`
	SessionID int  `json:"session_id"`
	Result    struct {
	} `json:"result"`
}

// OverviewResult softnas-cmd overview result for metrics
type OverviewResult struct {
	Success   bool `json:"success"`
	SessionID int  `json:"session_id"`
	Result    struct {
		Success bool   `json:"success"`
		Msg     string `json:"msg"`
		Records []struct {
			StorageName string  `json:"storage_name,omitempty"`
			StorageData float64 `json:"storage_data,omitempty"`
			MemoryName  string  `json:"memory_name,omitempty"`
			MemoryData  float64 `json:"memory_data,omitempty"`
		} `json:"records"`
		Total int `json:"total"`
	} `json:"result"`
}

// PerfmonResult softnas-cmd perfmon result for metrics
type PerfmonResult struct {
	Result struct {
		Msg     string `json:"msg"`
		Records []struct {
			ArcHitpercent int     `json:"arc_hitpercent"`
			ArcHits       int     `json:"arc_hits"`
			ArcMiss       int     `json:"arc_miss"`
			ArcRead       int     `json:"arc_read"`
			ArcSize       float64 `json:"arc_size"`
			ArcTarget     float64 `json:"arc_target"`
			CPU           float64 `json:"cpu"`
			IoDiskreads   int     `json:"io_diskreads"`
			IoDiskwrites  int     `json:"io_diskwrites"`
			IoNetreads    int     `json:"io_netreads"`
			IoNetwrites   int     `json:"io_netwrites"`
			IopsCifs      int     `json:"iops_cifs"`
			IopsIscsi     int     `json:"iops_iscsi"`
			IopsNfs       int     `json:"iops_nfs"`
			LatencyCifs   int     `json:"latency_cifs"`
			LatencyIscsi  int     `json:"latency_iscsi"`
			LatencyNfs    int     `json:"latency_nfs"`
			Time          string  `json:"time"`
		} `json:"records"`
		Success bool `json:"success"`
		Total   int  `json:"total"`
	} `json:"result"`
	SessionID int  `json:"session_id"`
	Success   bool `json:"success"`
}

// PoolDetailsResult softnas-cmd pooldetails result for metrics
type PoolDetailsResult struct {
	Result struct {
		Msg     string `json:"msg"`
		Records []struct {
			ChecksumErrors string `json:"checksum_errors"`
			Extended       string `json:"extended"`
			Name           string `json:"name"`
			ReadIOPS       string `json:"read_IOPS"`
			ReadBandwidth  string `json:"read_bandwidth"`
			ReadErrors     string `json:"read_errors"`
			Scrub          string `json:"scrub"`
			Status         string `json:"status"`
			WriteIOPS      string `json:"write_IOPS"`
			WriteBandwidth string `json:"write_bandwidth"`
			WriteErrors    string `json:"write_errors"`
		} `json:"records"`
		Success bool `json:"success"`
		Total   int  `json:"total"`
	} `json:"result"`
	SessionID int  `json:"session_id"`
	Success   bool `json:"success"`
}

// FetchMetrics interface for mackerelplugin
func (s SoftnasPlugin) FetchMetrics() (map[string]interface{}, error) {
	stat, err := s.parseStats()
	if err != nil {
		return nil, err
	}
	return stat, nil
}

// GraphDefinition interface for mackerel plugin
func (s SoftnasPlugin) GraphDefinition() map[string](mp.Graphs) {
	var graphdef = map[string](mp.Graphs){
		"softnas.storagename": mp.Graphs{
			Label: "SoftNas Storage Size",
			Unit:  "bytes",
			Metrics: [](mp.Metrics){
				mp.Metrics{Name: "storagename_used", Label: "Used", Diff: false},
				mp.Metrics{Name: "storagename_free", Label: "Free", Diff: false},
			},
		},
		"softnas.storagedata": mp.Graphs{
			Label: "SoftNas Storage Usage",
			Unit:  "percentage",
			Metrics: [](mp.Metrics){
				mp.Metrics{Name: "storagedata_used", Label: "Used", Diff: false},
				mp.Metrics{Name: "storagedata_free", Label: "Free", Diff: false},
			},
		},
		"softnas.memoryname": mp.Graphs{
			Label: "SoftNas Cache Memory Size",
			Unit:  "bytes",
			Metrics: [](mp.Metrics){
				mp.Metrics{Name: "memoryname_used", Label: "Used", Diff: false, Stacked: true},
				mp.Metrics{Name: "memoryname_free", Label: "Free", Diff: false, Stacked: true},
			},
		},
		"softnas.memorydata": mp.Graphs{
			Label: "SoftNas Cache Memory Usage",
			Unit:  "percentage",
			Metrics: [](mp.Metrics){
				mp.Metrics{Name: "memorydata_used", Label: "Used", Diff: false, Stacked: true},
				mp.Metrics{Name: "memorydata_free", Label: "Free", Diff: false, Stacked: true},
			},
		},
		"softnas.numberofarccache": mp.Graphs{
			Label: "SoftNas ARC Cache",
			Unit:  "float",
			Metrics: [](mp.Metrics){
				mp.Metrics{Name: "arc_hits", Label: "Hits", Diff: false},
				mp.Metrics{Name: "arc_miss", Label: "Miss", Diff: false},
				mp.Metrics{Name: "arc_read", Label: "Read", Diff: false},
			},
		},
		"softnas.pooliops": mp.Graphs{
			Label: "SoftNas Read/Write Pool IOPS",
			Unit:  "iops",
			Metrics: [](mp.Metrics){
				mp.Metrics{Name: "pool1_read_iops", Label: "Read_IOPS", Diff: false},
				mp.Metrics{Name: "pool1_write_iops", Label: "Write_IOPS", Diff: false},
			},
		},
	}
	return graphdef
}

//Get to convert the StorageName & MemoryName
func getSizeConvert(name string) (float64, error) {
	if strings.Contains(name, ",") {
		name = strings.Replace(name, ",", "", -1)
	}
	if strings.HasSuffix(name, "K") {
		nameConv, err := strconv.ParseFloat(strings.Trim(name, "K"), 64)
		return nameConv * 1024, err
	} else if strings.HasSuffix(name, "M") {
		nameConv, err := strconv.ParseFloat(strings.Trim(name, "M"), 64)
		return nameConv * 1024 * 1024, err
	} else if strings.HasSuffix(name, "G") {
		nameConv, err := strconv.ParseFloat(strings.Trim(name, "G"), 64)
		return nameConv * 1024 * 1024 * 1024, err
	} else if strings.HasSuffix(name, "T") {
		nameConv, err := strconv.ParseFloat(strings.Trim(name, "T"), 64)
		return nameConv * 1024 * 1024 * 1024 * 1024, err
	} else {
		nameConv, err := strconv.ParseFloat(name, 64)
		return nameConv, err
	}
}

func getMetricsAverage(mets []float64) float64 {
	var sum float64
	i := 0
	for _, met := range mets {
		if met != 0.0 {
			sum += met
			i++
		}
	}
	if sum == 0.0 {
		return 0.0
	}
	return sum / float64(i)
}

//Get the session_id of softnas-cmd
func getSoftnasSessionID(cmd string, url string, user string, pw string) (int, error) {
	var l LoginResult
	result, err := exec.Command(cmd, "login", user, pw, "--base_url", url).Output()
	json.Unmarshal([]byte(result), &l)
	return l.SessionID, err
}

func (s *SoftnasPlugin) parseStats() (map[string]interface{}, error) {
	var o OverviewResult
	var p PerfmonResult
	stat := make(map[string]interface{})
	oRes, err := exec.Command(s.Command, "overview", "--session_id", s.SessionID, "--base_url", s.BaseURL).Output()
	if err != nil {
		return nil, err
	}
	pRes, err := exec.Command(s.Command, "perfmon", "--session_id", s.SessionID, "--base_url", s.BaseURL).Output()
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(oRes), &o)
	json.Unmarshal([]byte(pRes), &p)

	//Parse StorageName&StrageData Metrics
	stat["storagename_free"], err = getSizeConvert(strings.Split(o.Result.Records[0].StorageName, " ")[0])
	if err != nil {
		return nil, err
	}
	stat["storagename_used"], err = getSizeConvert(strings.Split(o.Result.Records[1].StorageName, " ")[0])
	if err != nil {
		return nil, err
	}
	stat["storagedata_free"] = o.Result.Records[0].StorageData
	stat["storagedata_used"] = o.Result.Records[1].StorageData

	//Parse MemoryName&MemoryData Metrics
	stat["memoryname_free"], err = getSizeConvert(strings.Split(o.Result.Records[3].MemoryName, "\n")[0])
	if err != nil {
		return nil, err
	}
	stat["memoryname_used"], err = getSizeConvert(strings.Split(o.Result.Records[2].MemoryName, "\n")[0])
	if err != nil {
		return nil, err
	}
	stat["memorydata_free"] = o.Result.Records[3].MemoryData
	stat["memorydata_used"] = o.Result.Records[2].MemoryData

	//Parse NumberOfArcCache Metrics (average of a minute)
	pTotal := p.Result.Total
	ahSlice := make([]float64, pTotal)
	amSlice := make([]float64, pTotal)
	arSlice := make([]float64, pTotal)
	for i := 0; i < pTotal; i++ {
		ahSlice = append(ahSlice, float64(p.Result.Records[i].ArcHits))
		amSlice = append(amSlice, float64(p.Result.Records[i].ArcMiss))
		arSlice = append(arSlice, float64(p.Result.Records[i].ArcRead))
	}
	stat["arc_hits"] = getMetricsAverage(ahSlice)
	stat["arc_miss"] = getMetricsAverage(amSlice)
	stat["arc_read"] = getMetricsAverage(arSlice)

	//Parse Pool IOPS Metrics
	pStats, err := s.getPoolIOPSStats()
	if err != nil {
		return nil, err
	}
	for k, v := range pStats {
		stat[k] = v
	}

	return stat, nil
}

func (s *SoftnasPlugin) getPoolIOPSStats() (map[string]interface{}, error) {
	var p PoolDetailsResult
	stat := make(map[string]interface{})
	pdRes, err := exec.Command(s.Command, "pooldetails", "--session_id", s.SessionID, "--base_url", s.BaseURL).Output()
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(pdRes), &p)
	pdTotal := p.Result.Total
	for i := 0; i < pdTotal; i++ {
		if i%2 == 0 {
			pn := p.Result.Records[i].Name
			stat[pn+"_read_iops"], err = strconv.ParseFloat(p.Result.Records[i].ReadIOPS, 64)
			if err != nil {
				return nil, err
			}
			stat[pn+"_write_iops"], err = strconv.ParseFloat(p.Result.Records[i].WriteIOPS, 64)
			if err != nil {
				return nil, err
			}
		}
	}
	return stat, err
}

func main() {
	var optCommand = flag.String("cmd", "/usr/local/bin/softnas-cmd", "Path of softnas-cmd")
	var optBaseURL = flag.String("url", "https://localhost/softnas", "URL of softnas-cmd")
	var optUser = flag.String("user", "softnas", "User of softnas-cmd")
	var optPassword = flag.String("password", "Pass4W0rd", "Password of softnas-cmd")
	flag.Parse()

	var softnas SoftnasPlugin
	softnas.Command = *optCommand
	softnas.BaseURL = *optBaseURL
	softnas.User = *optUser
	softnas.Password = *optPassword

	id, err := getSoftnasSessionID(*optCommand, *optBaseURL, *optUser, *optPassword)
	if err != nil {
		fmt.Println(err)
	}

	softnas.SessionID = strconv.Itoa(id)

	helper := mp.NewMackerelPlugin(softnas)

	helper.Run()

}
