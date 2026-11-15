package main

import (
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/korandiz/v4l"
)

func main() {
	flags := NewFlags()
	flags.Parse()
	flags.Print()

	var (
		config = Config{
			HostName:    flags.HostName,
			HostAddress: "192.168.10.7",
			HostPort:    "8554",
			Cameras:     make([]*CameraInfo, 0),
			startChan:   make(chan *Activity, 4),
			endChan:     make(chan *Activity, 4),
		}
		err error
	)

	exists := config.HasFile()
	if exists {
		log.Printf("existing configuration found")
		err = config.Load()
		if err != nil {
			log.Printf("Error loading configuration file %s. %s", ConfigName, err)
			return
		}
	}

	config.ConfigureAll(flags)

	if flags.Info || flags.Update {
		err = config.Save()
		if err != nil {
			log.Printf("Error saving configuration file %s. %s", ConfigName, err)
		}
	}

	if flags.Info {
		return
	}

	activities := NewActivities()

	for _, cameraInfo := range config.Cameras {
		activity := NewActivity(cameraInfo, config.HostAddress, config.HostPort)
		if activity == nil {
			log.Printf("stream not found '%s'", cameraInfo.DeviceName)
			continue
		}
		activities = append(activities, activity)
	}

	for _, activity := range activities {
		go activity.Run(config.startChan, config.endChan)
	}

	active := make(map[string]*Activity)

	defer func() {
		activities.CancelAll()
		log.Print("cancelled")
		for k := range active {
			log.Printf("active %s", k)
		}
	}()

	ticker := time.NewTicker(10 * time.Second)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	for {
		select {
		case sig := <-sigs:
			log.Printf("Signal: %v", sig)
			return

		case started := <-config.startChan:
			started.Camera.Active = true
			active[started.Path] = started
			log.Printf("signal from '%s' started", started.Path)

		case ended := <-config.endChan:
			ended.Camera.Active = false
			delete(active, ended.Path)
			log.Printf("signal from '%s' ended", ended.Path)

		case <-ticker.C:
			scanned := scanVideo(flags, &config, active)
			if len(scanned) > 0 {
				log.Printf("new %v", scanned)
			}

		default:
			time.Sleep(time.Millisecond * 5)
		}
	}
}

func scanVideo(flags *FFsFlags, config *Config, active map[string]*Activity) (scanned []string) {
	scanned = make([]string, 0)
	devices := v4l.FindDevices()
	for _, deviceInfo := range devices {
		_, ok := active[deviceInfo.Path]
		if !ok {
			scanned = append(scanned, deviceInfo.Path)
			cameraInfo, err := config.Configure(flags, deviceInfo)
			if err == nil {
				activity := NewActivity(cameraInfo, config.HostAddress, config.HostPort)
				go activity.Run(config.startChan, config.endChan)
			}
		}
	}
	return
}
