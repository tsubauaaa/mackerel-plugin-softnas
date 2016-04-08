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

var graphdef = map[string](mp.Graphs){
	"softnas.storagename": mp.Graphs{
		Label: "SoftNas StorageName",
		Unit:  "bytes",
		Metrics: [](mp.Metrics){
			mp.Metrics{Name: "storagename_used", Label: "Used", Diff: false},
			mp.Metrics{Name: "storagename_free", Label: "Free", Diff: false},
		},
	},
	"softnas.storagedata": mp.Graphs{
		Label: "SoftNas StorageData",
		Unit:  "percentage",
		Metrics: [](mp.Metrics){
			mp.Metrics{Name: "storagedata_used", Label: "Used", Diff: false},
			mp.Metrics{Name: "storagedata_free", Label: "Free", Diff: false},
		},
	},
	"softnas.memoryname": mp.Graphs{
		Label: "SoftNas MemoryName",
		Unit:  "bytes",
		Metrics: [](mp.Metrics){
			mp.Metrics{Name: "memoryname_used", Label: "Used", Diff: false},
			mp.Metrics{Name: "memoryname_free", Label: "Free", Diff: false},
		},
	},
	"softnas.memorydata": mp.Graphs{
		Label: "SoftNas MemoryData",
		Unit:  "percentage",
		Metrics: [](mp.Metrics){
			mp.Metrics{Name: "memorydata_used", Label: "Used", Diff: false},
			mp.Metrics{Name: "memorydata_free", Label: "Free", Diff: false},
		},
	},
	"softnas.numberofarccache": mp.Graphs{
		Label: "SoftNas NumberOfArcCache",
		Unit:  "float",
		Metrics: [](mp.Metrics){
			mp.Metrics{Name: "arc_hits", Label: "Hits", Diff: false},
			mp.Metrics{Name: "arc_miss", Label: "Miss", Diff: false},
			mp.Metrics{Name: "arc_read", Label: "Read", Diff: false},
		},
	},
}

// SoftnasPlugin mackerel plugin for softnas
type SoftnasPlugin struct {
	Command   string
	BaseURL   string
	User      string
	Password  string
	SessionID string
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
	sum, i := 0.0, 0
	for _, met := range mets {
		if met != 0.0 {
			fmt.Println("met:", met)
			sum += met
			i++
		}
	}
	if sum == 0.0 {
		return 0.0
	}
	fmt.Println("i:", i)
	return sum / float64(i)
}

//Get the session_id of softnas-cmd
func getSoftnasSessionID(cmd string, url string, user string, pw string) (int, error) {
	var l LoginResult
	result, err := exec.Command(cmd, "login", user, pw, "--base_url", url).Output()
	json.Unmarshal([]byte(result), &l)
	return l.SessionID, err
}

func (s SoftnasPlugin) parseStats() (map[string]interface{}, error) {
	var o OverviewResult
	var p PerfmonResult
	stat := make(map[string]interface{})
	ores, err := exec.Command(s.Command, "overview", "--session_id", s.SessionID, "--base_url", s.BaseURL).Output()
	if err != nil {
		return nil, err
	}
	pres, err := exec.Command(s.Command, "perfmon", "--session_id", s.SessionID, "--base_url", s.BaseURL).Output()
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(ores), &o)
	json.Unmarshal([]byte(pres), &p)

	//Parse StorageName&StrageData Metrics
	sn0 := o.Result.Records[0].StorageName
	sn1 := o.Result.Records[1].StorageName
	sdFree := o.Result.Records[0].StorageData
	sdUsed := o.Result.Records[1].StorageData
	sn0Split := strings.Split(sn0, " ")
	sn1Split := strings.Split(sn1, " ")
	snFree, err := getSizeConvert(sn0Split[0])
	if err != nil {
		return nil, err
	}
	snUsed, err := getSizeConvert(sn1Split[0])
	if err != nil {
		return nil, err
	}
	stat["storagename_used"] = snUsed
	stat["storagename_free"] = snFree
	stat["storagedata_used"] = sdUsed
	stat["storagedata_free"] = sdFree

	//Parse MemoryName&MemoryData Metrics
	mn0 := o.Result.Records[3].MemoryName
	mn1 := o.Result.Records[2].MemoryName
	mdFree := o.Result.Records[3].MemoryData
	mdUsed := o.Result.Records[2].MemoryData
	mn0Split := strings.Split(mn0, "\n")
	mn1Split := strings.Split(mn1, "\n")
	mnFree, err := getSizeConvert(mn0Split[0])
	if err != nil {
		return nil, err
	}
	mnUsed, err := getSizeConvert(mn1Split[0])
	if err != nil {
		return nil, err
	}
	stat["memoryname_used"] = mnUsed
	stat["memoryname_free"] = mnFree
	stat["memorydata_used"] = mdUsed
	stat["memorydata_free"] = mdFree

	//Parse NumberOfArcCache Metrics
	t := p.Result.Total
	ahSlice := make([]float64, t)
	amSlice := make([]float64, t)
	arSlice := make([]float64, t)
	for i := 0; i < t; i++ {
		ahSlice = append(ahSlice, float64(p.Result.Records[i].ArcHits))
		amSlice = append(amSlice, float64(p.Result.Records[i].ArcMiss))
		arSlice = append(arSlice, float64(p.Result.Records[i].ArcRead))
	}
	stat["arc_hits"] = getMetricsAverage(ahSlice)
	stat["arc_miss"] = getMetricsAverage(amSlice)
	stat["arc_read"] = getMetricsAverage(arSlice)

	return stat, nil
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
