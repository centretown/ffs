package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/centretown/v4l"
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
	Piped      bool   `json:"piped"`     // false
	Active     bool   `json:"-"`
	Loopback   bool   `json:"-"`
}

type Config struct {
	HostName    string        `json:"hostname"`
	HostAddress string        `json:"hostaddress"` // 192.168.10.7
	HostPort    string        `json:"hostport"`    // :8554
	Cameras     []*CameraInfo `json:"cameras"`
	startChan   chan *Activity
	endChan     chan *Activity
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

func (config *Config) FindInfo(deviceInfo *v4l.DeviceInfo) (*CameraInfo, bool) {
	for _, cam := range config.Cameras {
		if cam.Active {
			continue
		}

		if cam.DeviceName == deviceInfo.DeviceName {
			return cam, true
		}
	}
	return nil, false
}

const (
	UVCVideoDriver = "uvcvideo"
	V4L2LOOPBACK   = "v4l2 loopback"
)

func (config *Config) Configure(flags *FFsFlags, deviceInfo v4l.DeviceInfo) (cameraInfo *CameraInfo, err error) {
	var printerr = true
	defer func() {
		if err != nil && printerr {
			log.Printf("Configure: %v", err)
		}
	}()

	if flags.IsExcluded(deviceInfo.DeviceName) {
		err = fmt.Errorf("Device name '%s' has been excluded\n", deviceInfo.DeviceName)
		printerr = false
		return
	}
	cameraInfo, found := config.FindInfo(&deviceInfo)

	if !deviceInfo.Camera {
		err = fmt.Errorf("Not a camera %s\n", deviceInfo.Path)
		return
	}

	if deviceInfo.DriverName != UVCVideoDriver && deviceInfo.DriverName != V4L2LOOPBACK {
		err = fmt.Errorf("the '%s' driver at '%s' is not supported\n",
			deviceInfo.DriverName, deviceInfo.Path)
		return
	}

	cameraInfo.Loopback = deviceInfo.DriverName == V4L2LOOPBACK
	var device *v4l.Device
	device, err = v4l.Open(deviceInfo.Path)
	if err != nil {
		err = fmt.Errorf("unable to open device %s: %v\n", deviceInfo.Path, err)
		return
	}

	fmt.Print("update/add camera\n")

	if !found {
		cameraInfo = &CameraInfo{
			DeviceName: deviceInfo.DeviceName,
			Alias:      config.NewAlias(),
		}
		fmt.Printf("a new device '%s' has been added\n", deviceInfo.DeviceName)
	}

	cameraInfo.Path = deviceInfo.Path
	if cameraInfo.Width == 0 || cameraInfo.Height == 0 {
		cameraInfo.Width = 1280
		cameraInfo.Height = 720
	}

	if cameraInfo.FrameRate == 0 {
		cameraInfo.FrameRate = 30
	}

	if !found {
		config.Cameras = append(config.Cameras, cameraInfo)
	}

	if flags.Info {
		info(&deviceInfo, device)
	}
	return cameraInfo, nil
}

func (config *Config) ConfigureAll(flags *FFsFlags) {
	devices := v4l.FindDevices()
	for _, deviceInfo := range devices {
		log.Printf("%s:'%s'", deviceInfo.Path, deviceInfo.DeviceName)
		config.Configure(flags, deviceInfo)
	}
}

func info(deviceInfo *v4l.DeviceInfo, device *v4l.Device) (err error) {
	defer func() {
		if err != nil {
			log.Printf("Configure: %v", err)
		}
	}()

	var (
		configs  []v4l.DeviceConfig
		controls []v4l.ControlInfo
	)

	controls, err = device.ListControls()
	if err != nil {
		fmt.Printf("Error listing controls %s@%s: %v\n", deviceInfo.DeviceName, deviceInfo.Path, err)
	}

	if controls == nil {
		err = fmt.Errorf("Controls list invalid %s@%s: %v\n", deviceInfo.DeviceName, deviceInfo.Path, err)
		return
	}

	fmt.Printf("name: '%s' path: '%s' available controls:\n", deviceInfo.DeviceName, deviceInfo.Path)
	for _, control := range controls {
		value, err := device.GetControl(control.CID)
		if err != nil {
			err = fmt.Errorf("Error GetControl %s@%s: %v", deviceInfo.DeviceName, deviceInfo.Path, err)
			continue
		}

		fmt.Printf("%s: value = %d, min = %d, max = %d, default = %d, step = %d\n",
			control.Name, value, control.Min, control.Max, control.Default, control.Step)
	}
	fmt.Println()

	configs, err = device.ListConfigs()
	if err != nil {
		err = fmt.Errorf("Error listing configs %s@%s: %v\n", deviceInfo.DeviceName, deviceInfo.Path, err)
		return
	}

	fmt.Printf("%s@%s available configurations:\n", deviceInfo.DeviceName, deviceInfo.Path)
	for i, c := range configs {
		fmt.Printf("%d: %dx%d @%dfps %s\n", i, c.Width, c.Height, c.FPS.N, FourCC(c.Format))
	}

	if len(configs) > 0 {
		err = device.SetConfig(configs[0])
		if err != nil {
			err = fmt.Errorf("Error setting config: %v\n", err)
			return
		}

		err = device.TurnOn()
		if err != nil {
			err = fmt.Errorf("%s\n\tturning on: %v\n", deviceInfo.DeviceName, err)
			return
		}

		device.TurnOff()
		fmt.Printf("%s\n\tcurrently available\n", deviceInfo.DeviceName)
	}
	return nil
}

// ensure no dups in current list
func (config *Config) NewAlias() (alias string) {
	i := len(config.Cameras)
	last := i
	for {
		alias = fmt.Sprintf("%s%02d", config.HostName, i)
		for _, c := range config.Cameras {
			if c.Alias == alias {
				i++
				break
			}
		}
		if i == last {
			break
		}
	}
	return
}
