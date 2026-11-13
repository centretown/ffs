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

	var (
		config = Config{
			HostName:    flags.HostName,
			HostAddress: "192.168.10.7",
			HostPort:    "8554",
			Cameras:     make([]*CameraInfo, 0),
			startChan:   make(chan string, 4),
			endChan:     make(chan string, 4),
		}
		err error
	)

	if flags.Info {
		configureAll(flags, &config)
		err = config.Save()
		if err != nil {
			log.Printf("Error updating configuration file %s. %s", ConfigName, err)
		}
		return
	}

	exists := config.HasFile()
	if exists {
		log.Printf("existing configuration found")
		err = config.Load()
		if err != nil {
			log.Printf("Error loading configuration file %s. %s", ConfigName, err)
			return
		}
	} else {
		log.Printf("create new configuration")
		configureAll(flags, &config)
	}

	if flags.Update {
		exists := config.HasFile()
		err = config.Save()
		if err != nil {
			log.Printf("Error updating configuration file %s. %s", ConfigName, err)
		} else if exists {
			log.Print("Updated configuration file. ", ConfigName)
		} else {
			log.Print("Created configuration file. ", ConfigName)
		}
	}

	flags.Print()

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

	active := make(map[string]bool)

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
			log.Printf("signal from '%s' started", started)
			active[started] = true
		case ended := <-config.endChan:
			delete(active, ended)
			log.Printf("signal from '%s' ended", ended)
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

func scanVideo(flags *FFsFlags, config *Config, active map[string]bool) (scanned []string) {
	scanned = make([]string, 0)
	devices := v4l.FindDevices()
	for _, deviceInfo := range devices {
		_, ok := active[deviceInfo.Path]
		if !ok {
			scanned = append(scanned, deviceInfo.Path)
			cameraInfo := configure(flags, config, deviceInfo)
			activity := NewActivity(cameraInfo, config.HostAddress, config.HostPort)
			go activity.Run(config.startChan, config.endChan)
		}
	}
	return
}
