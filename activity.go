package main

import (
	"context"
	"fmt"
	"log"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type Activity struct {
	Path       string
	Camera     *CameraInfo
	Stream     *ffmpeg.Stream
	Context    context.Context
	CancelFunc context.CancelFunc
}

func NewActivity(cam *CameraInfo, hostAddress, hostPort string) (ffs *Activity) {

	ffs = &Activity{}

	inputArgs := ffmpeg.KwArgs{
		"f":            "v4l2",
		"framerate":    30,
		"video_size":   fmt.Sprintf("%dx%d", cam.Width, cam.Height),
		"input_format": "mjpeg",
	}
	/*
		ffmpeg -f v4l2 -framerate 30 -video_size 2560x1440 -input_format mjpeg \
		 -i /dev/video6 -c:v libx264 -pix_fmt yuv420p -preset ultrafast -b:v 600k \
		 -f rtsp rtsp:192.168.10.7:8554/cam5
	*/
	outputArgs := ffmpeg.KwArgs{
		"c:v":     "libx264",
		"pix_fmt": "yuv420p",
		"preset":  "ultrafast",
		"b:v":     "600k",
		"f":       "rtsp",
	}

	log.Println("Create ffmpeg stream")
	outName := fmt.Sprintf("rtsp:%s:%s/%s", hostAddress, hostPort, cam.Alias)
	stream := ffmpeg.Input(cam.Path, inputArgs).Output(outName, outputArgs)
	ffs.Context, ffs.CancelFunc = context.WithCancel(stream.Context)
	stream.Context = ffs.Context
	ffs.Stream = stream
	ffs.Path = cam.Path
	ffs.Camera = cam

	return
}

type Activities []*Activity

func NewActivities() Activities {
	return make([]*Activity, 0)
}

func (m Activities) CancelAll() {
	for _, activity := range m {
		activity.CancelFunc()
	}
}

func (activity *Activity) Run(startChan, endChan chan string) {
	startChan <- activity.Path
	err := activity.Stream.Run()
	if err != nil {
		log.Printf("path: '%s' error=%v", activity.Path, err)
	}
	endChan <- activity.Path
}
