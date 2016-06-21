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
	PoolNames []string
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

func convertUnit(name string) (float64, error) {
	if strings.Contains(name, ",") {
		name = strings.Replace(name, ",", "", -1)
	}
	switch {
	case strings.HasSuffix(name, "K"):
		nameConv, err := strconv.ParseFloat(strings.Trim(name, "K"), 64)
		if err != nil {
			return 0.0, err
		}
		return nameConv * 1024, nil
	case strings.HasSuffix(name, "M"):
		nameConv, err := strconv.ParseFloat(strings.Trim(name, "M"), 64)
		if err != nil {
			return 0.0, err
		}
		return nameConv * 1024 * 1024, nil
	case strings.HasSuffix(name, "G"):
		nameConv, err := strconv.ParseFloat(strings.Trim(name, "G"), 64)
		if err != nil {
			return 0.0, err
		}
		return nameConv * 1024 * 1024 * 1024, nil
	case strings.HasSuffix(name, "T"):
		nameConv, err := strconv.ParseFloat(strings.Trim(name, "T"), 64)
		if err != nil {
			return 0.0, err
		}
		return nameConv * 1024 * 1024 * 1024 * 1024, nil
	case strings.HasSuffix(name, "B"):
		nameConv, err := strconv.ParseFloat(strings.Trim(name, "B"), 64)
		if err != nil {
			return 0.0, err
		}
		return nameConv, nil
	default:
		nameConv, err := strconv.ParseFloat(name, 64)
		if err != nil {
			return 0.0, err
		}
		return nameConv, nil
	}
}

func culculateAverage(mets []float64) float64 {
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

func fetchSessionID(cmd, url, user, pw string) (int, error) {
	var l LoginResult
	result, err := exec.Command(cmd, "login", user, pw, "--base_url", url).Output()
	json.Unmarshal([]byte(result), &l)
	return l.SessionID, err
}

func fetchPoolNames(cmd, id, url string) ([]string, error) {
	var p PoolDetailsResult
	result, err := exec.Command(cmd, "pooldetails", "--session_id", id, "--base_url", url).Output()
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(result), &p)

	var pns []string
	for _, record := range p.Result.Records {
		if strings.Contains(record.Name, "&nbsp;") {
			continue
		}
		pns = append(pns, record.Name)
	}
	return pns, nil
}

func fetchOverviewMetrics(cmd, id, url string) (map[string]interface{}, error) {
	var o OverviewResult
	stat := make(map[string]interface{})
	result, err := exec.Command(cmd, "overview", "--session_id", id, "--base_url", url).Output()
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(result), &o)

	//Parse StorageName&StrageData Metrics
	stat["storagename_free"], err = convertUnit(strings.Split(o.Result.Records[0].StorageName, " ")[0])
	if err != nil {
		return nil, err
	}
	stat["storagename_used"], err = convertUnit(strings.Split(o.Result.Records[1].StorageName, " ")[0])
	if err != nil {
		return nil, err
	}
	stat["storagedata_free"] = o.Result.Records[0].StorageData
	stat["storagedata_used"] = o.Result.Records[1].StorageData

	//Parse MemoryName&MemoryData Metrics
	stat["memoryname_free"], err = convertUnit(strings.Split(o.Result.Records[3].MemoryName, "\n")[0])
	if err != nil {
		return nil, err
	}
	stat["memoryname_used"], err = convertUnit(strings.Split(o.Result.Records[2].MemoryName, "\n")[0])
	if err != nil {
		return nil, err
	}
	stat["memorydata_free"] = o.Result.Records[3].MemoryData
	stat["memorydata_used"] = o.Result.Records[2].MemoryData

	return stat, nil
}

func fetchPerfMonMetrics(cmd, id, url string) (map[string]interface{}, error) {
	var p PerfmonResult
	stat := make(map[string]interface{})
	result, err := exec.Command(cmd, "perfmon", "--session_id", id, "--base_url", url).Output()
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(result), &p)

	total := p.Result.Total
	ahSlice := make([]float64, total)
	amSlice := make([]float64, total)
	arSlice := make([]float64, total)
	for _, record := range p.Result.Records {
		ahSlice = append(ahSlice, float64(record.ArcHits))
		amSlice = append(amSlice, float64(record.ArcMiss))
		arSlice = append(arSlice, float64(record.ArcRead))
	}
	stat["arc_hits"] = culculateAverage(ahSlice)
	stat["arc_miss"] = culculateAverage(amSlice)
	stat["arc_read"] = culculateAverage(arSlice)

	return stat, nil
}

func fetchPoolIOPSMetrics(cmd, id, url string, poolNames []string) (map[string]interface{}, error) {
	var p PoolDetailsResult
	stat := make(map[string]interface{})
	for _, pn := range poolNames {
		result, err := exec.Command(cmd, "pooldetails", pn, "--session_id", id, "--base_url", url).Output()
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(result), &p)
		stat[pn+"_read_iops"], err = strconv.ParseFloat(p.Result.Records[0].ReadIOPS, 64)
		if err != nil {
			return nil, err
		}
		stat[pn+"_write_iops"], err = strconv.ParseFloat(p.Result.Records[0].WriteIOPS, 64)
		if err != nil {
			return nil, err
		}
	}
	return stat, nil
}

func mergeStats(dst, src map[string]interface{}) {
	for k, v := range src {
		dst[k] = v
	}
}

// FetchMetrics interface for mackerelplugin
func (s SoftnasPlugin) FetchMetrics() (map[string]interface{}, error) {
	oStats, err := fetchOverviewMetrics(s.Command, s.SessionID, s.BaseURL)
	if err != nil {
		return nil, err
	}
	pmStats, err := fetchPerfMonMetrics(s.Command, s.SessionID, s.BaseURL)
	if err != nil {
		return nil, err
	}
	pStats, err := fetchPoolIOPSMetrics(s.Command, s.SessionID, s.BaseURL, s.PoolNames)
	if err != nil {
		return nil, err
	}
	stat := make(map[string]interface{})

	mergeStats(stat, oStats)
	mergeStats(stat, pmStats)
	mergeStats(stat, pStats)

	return stat, nil
}

// GraphDefinition interface for mackerel plugin
func (s SoftnasPlugin) GraphDefinition() map[string](mp.Graphs) {
	metrics := make([](mp.Metrics), 0, len(s.PoolNames))
	for _, pn := range s.PoolNames {
		rm := mp.Metrics{Name: pn + "_read_iops"}
		rm.Label = strings.ToUpper(pn) + "_Read_IOPS"
		rm.Diff = false
		metrics = append(metrics, rm)

		wm := mp.Metrics{Name: pn + "_write_iops"}
		wm.Label = strings.ToUpper(pn) + "_Write_IOPS"
		wm.Diff = false
		metrics = append(metrics, wm)
	}

	var graphdef = map[string](mp.Graphs){
		"softnas.storagename": mp.Graphs{
			Label: "SoftNas Storage Size",
			Unit:  "bytes",
			Metrics: [](mp.Metrics){
				mp.Metrics{Name: "storagename_used", Label: "Used", Diff: false, Stacked: true},
				mp.Metrics{Name: "storagename_free", Label: "Free", Diff: false, Stacked: true},
			},
		},
		"softnas.storagedata": mp.Graphs{
			Label: "SoftNas Storage Usage",
			Unit:  "percentage",
			Metrics: [](mp.Metrics){
				mp.Metrics{Name: "storagedata_used", Label: "Used", Diff: false, Stacked: true},
				mp.Metrics{Name: "storagedata_free", Label: "Free", Diff: false, Stacked: true},
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
			Label:   "SoftNas Read/Write Pool IOPS",
			Unit:    "iops",
			Metrics: metrics,
		},
	}
	return graphdef
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

	id, err := fetchSessionID(*optCommand, *optBaseURL, *optUser, *optPassword)
	if err != nil {
		fmt.Println(err)
	}

	softnas.SessionID = strconv.Itoa(id)

	pns, err := fetchPoolNames(*optCommand, softnas.SessionID, *optBaseURL)
	if err != nil {
		fmt.Println(err)
	}

	softnas.PoolNames = pns

	helper := mp.NewMackerelPlugin(softnas)

	helper.Run()

}
