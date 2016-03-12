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

var graphdefSoftnas = map[string](mp.Graphs){
	"softnas.storagename": mp.Graphs{
		Label: "Softnas StorageName Used/Free",
		Unit:  "bytes",
		Metrics: [](mp.Metrics){
			mp.Metrics{Name: "storagename_used", Label: "Used", Diff: false},
			mp.Metrics{Name: "storagename_free", Label: "Free", Diff: false},
		},
	},
	"softnas.storagedata": mp.Graphs{
		Label: "Softnas StorageData Used/Free",
		Unit:  "percentage",
		Metrics: [](mp.Metrics){
			mp.Metrics{Name: "storagedata_used", Label: "Used", Diff: false},
			mp.Metrics{Name: "storagedata_free", Label: "Free", Diff: false},
		},
	},
}

// SoftnasPlugin mackerel plugin for softnas
type SoftnasPlugin struct {
	Command   string
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

// FetchMetrics interface for mackerelplugin
func (s SoftnasPlugin) FetchMetrics() (map[string]interface{}, error) {
	stat, err := s.getSoftnasOverview()
	if err != nil {
		return nil, err
	}
	return stat, err
}

// GraphDefinition interface for mackerel plugin
func (s SoftnasPlugin) GraphDefinition() map[string](mp.Graphs) {
	return graphdefSoftnas
}

//storagenameをByte変換する
func byteSizeConvert(storagename string) (float64, error) {
	if strings.HasSuffix(storagename, "K") {
		storagenameConv, err := strconv.ParseFloat(strings.Trim(storagename, "K"), 64)
		if err != nil {
			return 0, err
		}
		return storagenameConv * 1024, nil
	} else if strings.HasSuffix(storagename, "M") {
		storagenameConv, err := strconv.ParseFloat(strings.Trim(storagename, "M"), 64)
		if err != nil {
			return 0, err
		}
		return storagenameConv * 1024 * 1024, nil
	} else if strings.HasSuffix(storagename, "G") {
		storagenameConv, err := strconv.ParseFloat(strings.Trim(storagename, "G"), 64)
		if err != nil {
			return 0, err
		}
		return storagenameConv * 1024 * 1024 * 1024, nil
	} else if strings.HasSuffix(storagename, "T") {
		storagenameConv, err := strconv.ParseFloat(strings.Trim(storagename, "T"), 64)
		if err != nil {
			return 0, err
		}
		return storagenameConv * 1024 * 1024 * 1024 * 1024, nil
	} else {
		storagenameConv, err := strconv.ParseFloat(storagename, 64)
		if err != nil {
			return 0, err
		}
		return storagenameConv, nil
	}
}

//softnas-cmdのsession_idを取得
func getSoftnasSessionID(cmd string, user string, pw string) (int, error) {
	var login LoginResult
	result, err := exec.Command(cmd, "login", user, pw).Output()
	if err != nil {
		return 0, err
	}
	json.Unmarshal([]byte(result), &login)
	return login.SessionID, nil
}

//
func (s SoftnasPlugin) getSoftnasOverview() (map[string]interface{}, error) {
	var overview OverviewResult
	stat := make(map[string]interface{})
	result, err := exec.Command(s.Command, "--session_id", s.SessionID, "overview").Output()
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(result), &overview)
	storagename0 := overview.Result.Records[0].StorageName
	storagename1 := overview.Result.Records[1].StorageName
	storagedataFree := overview.Result.Records[0].StorageData
	storagedataUsed := overview.Result.Records[1].StorageData
	storagename0Split := strings.Split(storagename0, " ")
	storagename1Split := strings.Split(storagename1, " ")
	storagenameFree, err := byteSizeConvert(storagename0Split[0])
	storagenameUsed, err := byteSizeConvert(storagename1Split[0])
	stat["storagename_used"] = storagenameUsed
	stat["storagename_free"] = storagenameFree
	stat["storagedata_used"] = storagedataUsed
	stat["storagedata_free"] = storagedataFree
	return stat, err
}

func main() {
	// (name, default, help)
	var optCommand = flag.String("cmd", "/usr/local/bin/softnas-cmd", "Path of softnas-cmd")
	var optUser = flag.String("user", "softnas", "User of softnas-cmd")
	var optPassword = flag.String("password", "Pass4W0rd", "Password of softnas-cmd")
	flag.Parse()

	var softnas SoftnasPlugin
	softnas.Command = *optCommand
	softnas.User = *optUser
	softnas.Password = *optPassword

	id, err := getSoftnasSessionID(*optCommand, *optUser, *optPassword)
	if err != nil {
		fmt.Println(err)
	}

	softnas.SessionID = strconv.Itoa(id)

	helper := mp.NewMackerelPlugin(softnas)

	helper.Run()

}
