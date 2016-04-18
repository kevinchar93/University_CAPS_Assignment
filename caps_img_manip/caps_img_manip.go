package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

func check(e error) {
	if e != nil {
		fmt.Print(e.Error())
		panic(e)
	}
}

type empty struct{}
type semaphore chan empty

var methodFlag string
var breakdownFlag string
var progressFlag bool
var cpuCountFlag int

func main() {
	initFlags()

	var cpuCount int

	// get the ammount of CPU's to use
	if cpuCountFlag == 0 {
		cpuCount = 1
	} else if cpuCountFlag > runtime.NumCPU() || cpuCountFlag == -1 {
		cpuCount = runtime.NumCPU()
	} else {
		cpuCount = cpuCountFlag
	}
	runtime.GOMAXPROCS(cpuCount)

	// begin the execution timer
	startTime := time.Now()

	if len(flag.Args()) == 0 {
		fmt.Print("no arguments given to search\n")
		return
	} else if breakdownFlag == "pixel" {

		if methodFLag == "para" {

			// perfrom operation in parallel
		} else if methodFLag == "seq" {

			// perform operation sequentially
		} else {
			fmt.Print("Invalide method supplied\n")
			return
		}

	} else if breakdownFlag == "line" {

		if methodFLag == "para" {

			// perfrom operation in parallel
		} else if methodFLag == "seq" {

			// perform operation sequentially
		} else {
			fmt.Print("Invalide method supplied\n")
			return
		}
	} else {
		fmt.Print("Invalide breakdown supplied\n")
		return
	}

	elasped := time.Since(startTime)
	elaspedInSeconds := elasped.Seconds()

	// print results
	fmt.Print("\nOperation complete\n")
	fmt.Print("-----------------------------------------------\n")

	fmt.Print("\nSummary\n")
	fmt.Print("-----------------------------------------------\n")
}

func initFlags() {
	flag.StringVar(&filter, "breakdown", "pixel", "operate on the image pixel by pixel or line by line")
	flag.StringVar(&methodFlag, "method", "seq", "modify the image using either sequential or parrallel method")
	flag.BoolVar(&progressFlag, "progress", false, "shows the progress of the algorithm or not (T/F)")
	flag.IntVar(&cpuCountFlag, "cpus", -1, "how many cpus to use")

	flag.Parse()
}
