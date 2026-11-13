package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

type FFsFlags struct {
	HostName string
	Exclude  StringArray
	Update   bool
	Info     bool
}

func NewFlags() (ffsf *FFsFlags) {
	ffsf = &FFsFlags{}
	*ffsf = defaultFlags
	hostname, err := os.Hostname()
	if err == nil {
		ffsf.HostName = hostname
		return
	} else {
		fmt.Println("hostname:", err)
	}
	return
}

var (
	defaultFlags = FFsFlags{
		Exclude: make([]string, 0),
		Update:  false,
	}

	hostNameUsage = "name to use aa prefix"
	updateUsage   = "update default values"
	excludeUsage  = "cameras to exclude"
	infoUsage     = "display camera information and exit"
)

func (ffsf *FFsFlags) Print() {
	fmt.Printf("Host: %s\n", ffsf.HostName)
	fmt.Printf("Include:\n")
	for _, adr := range ffsf.Exclude {
		fmt.Printf("- %s\n", adr)
	}
	fmt.Printf("Update default values: %v\n", ffsf.Update)
}

func (ffsf *FFsFlags) Parse() {
	flag.StringVar(&ffsf.HostName, "host", ffsf.HostName, hostNameUsage)
	flag.StringVar(&ffsf.HostName, "h", ffsf.HostName, hostNameUsage)
	flag.BoolVar(&ffsf.Update, "update", ffsf.Update, updateUsage)
	flag.BoolVar(&ffsf.Update, "u", ffsf.Update, updateUsage)
	flag.BoolVar(&ffsf.Info, "info", ffsf.Info, infoUsage)
	flag.BoolVar(&ffsf.Info, "i", ffsf.Info, infoUsage)
	flag.Var(&ffsf.Exclude, "exclude", excludeUsage)
	flag.Var(&ffsf.Exclude, "x", excludeUsage)
	flag.Parse()
}

func (ffsf *FFsFlags) SetDefault() { *ffsf = defaultFlags }

func (ffsf *FFsFlags) IsExcluded(name string) bool {
	for _, excluded := range ffsf.Exclude {
		if strings.Contains(name, excluded) {
			log.Printf("'%s' is excluded '%s'", name, excluded)
			return true
		}
	}
	return false
}
