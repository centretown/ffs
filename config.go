package main

import (
	"encoding/json"
	"fmt"
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

const UVCVideoDriver = "uvcvideo"

func (config *Config) Configure(flags *FFsFlags, deviceInfo v4l.DeviceInfo) *CameraInfo {
	if flags.IsExcluded(deviceInfo.DeviceName) {
		fmt.Printf("Device '%s' has been excluded\n", deviceInfo.DeviceName)
		return nil
	}

	cameraInfo := config.FindInfo(&deviceInfo)
	configured := cameraInfo != nil
	if !configured {
		cameraInfo = &CameraInfo{
			DeviceName: deviceInfo.DeviceName,
		}
		fmt.Printf("Device '%s' has been created\n", deviceInfo.DeviceName)
	}

	cameraInfo.Path = deviceInfo.Path

	fmt.Print(deviceInfo.Path, "...")
	if !deviceInfo.Camera {
		fmt.Printf("Not a camera %s\n", deviceInfo.Path)
		return nil
	}

	fmt.Printf("Uses '%s' driver camera %s\n",
		deviceInfo.DriverName,
		deviceInfo.Path)

	if deviceInfo.DriverName != UVCVideoDriver {
		return nil
	}

	device, err := v4l.Open(deviceInfo.Path)
	if err != nil {
		fmt.Printf("Error opening device %s: %v\n",
			deviceInfo.Path, err)
		return nil
	}

	fmt.Print("update/add camera\n")

	if len(cameraInfo.Alias) < 1 {
		cameraInfo.Alias = fmt.Sprintf("%s%02d",
			config.HostName, len(config.Cameras))
	}

	if cameraInfo.Width == 0 || cameraInfo.Height == 0 {
		cameraInfo.Width = 1920
		cameraInfo.Height = 1080
	}

	if cameraInfo.FrameRate == 0 {
		cameraInfo.FrameRate = 30
	}

	if !configured {
		config.Cameras = append(config.Cameras, cameraInfo)
	}
	if flags.Info {
		info(&deviceInfo, device)
	}
	return cameraInfo
}

func (config *Config) ConfigureAll(flags *FFsFlags) {
	devices := v4l.FindDevices()
	for _, deviceInfo := range devices {
		log.Printf("%s:'%s'", deviceInfo.Path, deviceInfo.DeviceName)
		config.Configure(flags, deviceInfo)
	}
}

func info(deviceInfo *v4l.DeviceInfo, device *v4l.Device) {

	var (
		err      error
		configs  []v4l.DeviceConfig
		controls []v4l.ControlInfo
	)

	controls, err = device.ListControls()
	if err != nil {
		fmt.Printf("Error listing controls %s@%s: %v\n", deviceInfo.DeviceName, deviceInfo.Path, err)
	}

	if controls == nil {
		fmt.Printf("Controls list invalid %s@%s: %v\n", deviceInfo.DeviceName, deviceInfo.Path, err)
		return
	}

	fmt.Printf("name: '%s' path: '%s' available controls:\n", deviceInfo.DeviceName, deviceInfo.Path)
	for _, control := range controls {
		value, err := device.GetControl(control.CID)
		if err != nil {
			fmt.Printf("Error GetControl %s@%s: %v", deviceInfo.DeviceName, deviceInfo.Path, err)
			continue
		}

		fmt.Printf("%s: value = %d, min = %d, max = %d, default = %d, step = %d\n",
			control.Name, value, control.Min, control.Max, control.Default, control.Step)
	}
	fmt.Println()

	configs, err = device.ListConfigs()
	if err != nil {
		fmt.Printf("Error listing configs %s@%s: %v\n", deviceInfo.DeviceName, deviceInfo.Path, err)
		return
	}

	fmt.Printf("%s@%s available configurations:\n", deviceInfo.DeviceName, deviceInfo.Path)
	for i, c := range configs {
		fmt.Printf("%d: %dx%d @%dfps %s\n", i, c.Width, c.Height, c.FPS.N, FourCC(c.Format))
	}

	if len(configs) > 0 {
		err := device.SetConfig(configs[0])
		if err != nil {
			fmt.Printf("Error setting config: %v\n", err)
			return
		}

		err = device.TurnOn()
		if err != nil {
			fmt.Printf("%s\n\tturning on: %v\n", deviceInfo.DeviceName, err)
			return
		}

		device.TurnOff()
		fmt.Printf("%s\n\tcurrently available\n", deviceInfo.DeviceName)
	}
}
