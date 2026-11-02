package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
)

type stringArray []string

var _ flag.Value = (*stringArray)(nil)

// String is an implementation of the flag.Value interface
func (sa *stringArray) String() string {
	return fmt.Sprintf("%v", *sa)
}

// Set is an implementation of the flag.Value interface
func (sa *stringArray) Set(value string) error {
	*sa = append(*sa, value)
	return nil
}

type FFsFlags struct {
	HostAddr string
	Include  []string
	Exclude  []string
	Update   bool
}

func NewFFsFlags() (ffsFlags *FFsFlags) {
	ffsFlags = &FFsFlags{}
	*ffsFlags = avDefaultFlags
	return
}

const (
	ConfigName = "ffs.json"
)

var (
	avDefaultFlags = FFsFlags{
		Include: make([]string, 0),
		Exclude: make([]string, 0),
		Update:  false,
	}

	connectUsage  = "remote connections (none,all,restrict)"
	hostAddrUsage = "host ip address"
	updateUsage   = "update default values"
)

func (ffsFlags *FFsFlags) Print() {
	fmt.Printf("Host: %s\n", ffsFlags.HostAddr)
	fmt.Printf("Remotes:\n")
	for _, adr := range ffsFlags.Include {
		fmt.Printf("- %s\n", adr)
	}
	fmt.Printf("Update default values: %v\n", ffsFlags.Update)
}

func (ffsFlags *FFsFlags) Parse() {
	flag.StringVar(&ffsFlags.HostAddr, "address", ffsFlags.HostAddr, hostAddrUsage)
	flag.StringVar(&ffsFlags.HostAddr, "a", ffsFlags.HostAddr, hostAddrUsage)
	flag.BoolVar(&ffsFlags.Update, "update", ffsFlags.Update, updateUsage)
	flag.BoolVar(&ffsFlags.Update, "u", ffsFlags.Update, updateUsage)

	flag.Parse()

	if ffsFlags.Update {
		exists := ffsFlags.HasFile()
		err := ffsFlags.Save()
		if err != nil {
			log.Printf("Error updating configuration file %s. %s", ConfigName, err)
		} else if exists {
			log.Print("Updated configuration file. ", ConfigName)
		} else {
			log.Print("Created configuration file. ", ConfigName)
		}
	}

}

func (ffsFlags *FFsFlags) HasFile() bool {
	_, err := os.Stat(ConfigName)
	return err == nil
}

func (ffsFlags *FFsFlags) Load() (err error) {
	var buf []byte

	buf, err = os.ReadFile(ConfigName)
	if err != nil {
		log.Printf("ffsFlags Load ReadFile error: %s", err)
		return
	}

	err = json.Unmarshal(buf, ffsFlags)
	if err != nil {
		log.Printf("ffsFlags Load Unmarshal error: %s", err)
		return
	}

	return
}

func (ffsFlags *FFsFlags) Save() (err error) {
	var buf []byte
	buf, err = json.MarshalIndent(ffsFlags, "", "  ")
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

func (ffsFlags *FFsFlags) SetDefault() { *ffsFlags = avDefaultFlags }
