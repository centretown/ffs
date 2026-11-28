package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/centretown/v4l"
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
	var (
		inputArgs ffmpeg.KwArgs
		inputPath = cam.Path
	)

	if cam.Piped {
		inputPath = "pipe:"
		inputArgs = ffmpeg.KwArgs{
			"format":    "jpeg_pipe",
			"pix_fmt":   "yuv420p",
			"framerate": 30,
			"s":         fmt.Sprintf("%dx%d", cam.Width, cam.Height),
		}
	} else {
		inputArgs = ffmpeg.KwArgs{
			"f":          "v4l2",
			"framerate":  30,
			"video_size": fmt.Sprintf("%dx%d", cam.Width, cam.Height),
		}
		if !cam.Loopback {
			inputArgs["input_format"] = "mjpeg"
		}
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
	stream := ffmpeg.Input(inputPath, inputArgs).Output(outName, outputArgs)
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

func (activity *Activity) Run(startChan, endChan chan *Activity) {
	startChan <- activity

	if activity.Camera.Piped {
		done := make(chan int)
		reader, writer := io.Pipe()
		go activity.Pipe(writer, done)
		activity.Stream = activity.Stream.WithInput(reader)
		err := activity.Stream.Run()
		if err != nil {
			log.Printf("path: '%s' error=%v", activity.Path, err)
		}
		done <- 1
	} else {
		err := activity.Stream.Run()
		if err != nil {
			log.Printf("path: '%s' error=%v", activity.Path, err)
		}
	}

	endChan <- activity
}

func (activity *Activity) Pipe(writer *io.PipeWriter, done chan int) {

	var (
		err     error
		device  *v4l.Device
		buf     *v4l.Buffer
		bufInfo v4l.BufferInfo
		pixels  []byte
		count   int
	)

	defer func() {
		if err != nil {
			log.Fatal(err)
		}
	}()

	device, err = v4l.Open(activity.Path)
	if err != nil {
		return
	}
	err = device.TurnOn()
	if err != nil {
		return
	}

	defer device.TurnOff()

	bufInfo, err = device.BufferInfo()
	pixels = make([]byte, bufInfo.BufferSize)
	log.Printf("buffer size %d", bufInfo.BufferSize)

	time.Sleep(time.Second * 1)

	for {
		// time.Sleep(time.Millisecond * 2)
		select {
		case <-done:
			err = writer.Close()
			return
		default:
			buf, err = device.Capture()
			if err != nil {
				return
			}
			count, err = buf.Read(pixels)
			if err == nil {
				_, err = writer.Write(pixels[:count])
			}

			if err != nil {
				log.Println("writePixels", err)
				return
			}
		}
	}
}
