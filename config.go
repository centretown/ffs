package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/korandiz/v4l"
)

const (
	ConfigName = "ffs.json"
)

type CameraInfo struct {
	DeviceName string `json:"devicename"`
	Alias      string `json:"alias"`     // cam5
	Path       string `json:"path"`      // /dev/video6
	Width      int    `json:"width"`     // 2560
	Height     int    `json:"height"`    // 1440
	FrameRate  int    `json:"framerate"` // 30
}

type Config struct {
	HostName    string        `json:"hostname"`
	HostAddress string        `json:"hostaddress"` // 192.168.10.7
	HostPort    string        `json:"hostport"`    // :8554
	Cameras     []*CameraInfo `json:"cameras"`
	startChan   chan string
	endChan     chan string
}

func (config *Config) HasFile() bool {
	_, err := os.Stat(ConfigName)
	return err == nil
}

func (config *Config) Load() (err error) {
	var buf []byte

	buf, err = os.ReadFile(ConfigName)
	if err != nil {
		log.Printf("ffsFlags Load ReadFile error: %s", err)
		return
	}

	err = json.Unmarshal(buf, config)
	if err != nil {
		log.Printf("ffsFlags Load Unmarshal error: %s", err)
		return
	}

	return
}

func (config *Config) Save() (err error) {
	var buf []byte
	buf, err = json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Printf("ffsFlags Save Marshall error: %s", err)
		return
	}

	err = os.WriteFile(ConfigName, buf, 0644)
	if err != nil {
		log.Printf("ffsFlags Save WriteFile error: %s", err)
		return
	}

	return
}

func (config *Config) FindInfo(deviceInfo *v4l.DeviceInfo) *CameraInfo {
	for _, cam := range config.Cameras {
		if cam.Path == deviceInfo.Path {
			return cam
		}

		if cam.DeviceName == deviceInfo.DeviceName {
			return cam
		}
	}
	return nil
}
