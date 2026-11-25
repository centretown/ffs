package main

import (
	"flag"
	"fmt"
	"slices"
)

type StringArray []string

var _ flag.Value = (*StringArray)(nil)

// String is an implementation of the flag.Value interface
func (sa *StringArray) String() string {
	return fmt.Sprintf("%v", *sa)
}

// Set is an implementation of the flag.Value interface
func (sa *StringArray) Set(value string) error {
	if !slices.Contains(*sa, value) {
		*sa = append(*sa, value)
	}
	return nil
}
