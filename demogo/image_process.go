package main

import (
	"flag"
	"fmt"
	"github.com/disintegration/imaging"
	"image"
	"image/color"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type floatingColor struct {
	R float64
	G float64
	B float64
}

type empty struct{}
type semaphore chan empty

var filterFlag string
var cpuCountFlag int
var selectedFilter []float64
var selectedFilterStr string
var fileName string

func main() {

	initFlags()
	setSelectedFilter()

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

	// open the image
	srcImg, err := imaging.Open(fileName)
	imagePixelCount := srcImg.Bounds().Max.X * srcImg.Bounds().Max.Y
	check(err)

	// create a struct representation of the image
	srcImgNRGB := imaging.Clone(srcImg)

	// get the indexs to begin and end at
	beginX := srcImg.Bounds().Min.X
	beginY := srcImg.Bounds().Min.Y

	// crate a struct to put the new ouput image into
	outWidth := srcImg.Bounds().Max.X
	outHeight := srcImg.Bounds().Max.Y

	// alloc memory to put result images into
	outImage := image.NewNRGBA(image.Rect(0, 0, outWidth, outHeight))
	outImagePara := image.NewNRGBA(image.Rect(0, 0, outWidth, outHeight))

	// sequential operation ----------------------------------------------------
	startTimeSeq := time.Now()

	// iterate through each pixel stored in the NRGBA struct
	for rowY := beginY; rowY < outHeight; rowY++ {
		for pixelX := beginX; pixelX < outWidth; pixelX++ {
			applyKernelPixel(pixelX, rowY, srcImgNRGB, outImage, selectedFilter)
		}
	}

	// get operation metrics
	fmt.Print("") // needed of else timing does not work, don't know cause
	elaspedTimeSeq := time.Since(startTimeSeq)
	elaspedInSecondsSeq := elaspedTimeSeq.Seconds()
	pixelsPerSecondSeq := float64(imagePixelCount) / elaspedInSecondsSeq

	extension := filepath.Ext(fileName)
	nameNoExtension := fileName[0 : len(fileName)-len(extension)]
	outputName := fmt.Sprintf("%s_%s_output_seq.jpg", nameNoExtension, selectedFilterStr)
	err = imaging.Save(outImage, outputName)
	if err != nil {
		panic(err)
	}
	//--------------------------------------------------------------------------

	// parallel operation ------------------------------------------------------
	outImage = image.NewNRGBA(image.Rect(0, 0, outWidth, outHeight))
	var wg sync.WaitGroup // wait group to syncronise routines
	startTimePara := time.Now()

	// make a semaphore channel
	//runningProcesses := make(semaphore, 1)

	// iterate through each pixel stored in the NRGBA struct
	for rowY := beginY; rowY < outHeight; rowY++ {
		//go parallelApplyKernel(rowY, beginX, outWidth, srcImgNRGB, outImagePara, selectedFilter, &wg)
		go func(row, begin, end int, src *image.NRGBA, dest *image.NRGBA, kernel []float64, wg *sync.WaitGroup) {
			wg.Add(1)
			defer wg.Done()

			for pixel := begin; pixel < end; pixel++ {
				applyKernelPixel(pixel, row, src, dest, kernel)
			}
		}(rowY, beginX, outWidth, srcImgNRGB, outImagePara, selectedFilter, &wg)
	}
	wg.Wait()

	// get operation metrics
	fmt.Print("") // needed of else timing does not work, don't know cause
	elaspedTimePara := time.Since(startTimePara)
	elaspedInSecondsPara := elaspedTimePara.Seconds()
	pixelsPerSecondPara := float64(imagePixelCount) / elaspedInSecondsPara

	outputName = fmt.Sprintf("%s_%s_output_para.jpg", nameNoExtension, selectedFilterStr)
	err = imaging.Save(outImagePara, outputName)
	if err != nil {
		panic(err)
	}
	//--------------------------------------------------------------------------

	// operation report --------------------------------------------------------
	fmt.Print("\nSummary\n")
	fmt.Print("-----------------------------------------------\n")
	fmt.Print(fmt.Sprintf("Number CPUs: %d\n", cpuCount))
	fmt.Print(fmt.Sprintf("Filter operation: %s\n", selectedFilterStr))
	fmt.Print(fmt.Sprintf("Number of pixels: %d\n", imagePixelCount))
	fmt.Print("\nSequential operation\n")
	fmt.Print("---------------------\n")
	fmt.Print(fmt.Sprintf("Time elapsed: %s\n", elaspedTimeSeq))
	fmt.Print(fmt.Sprintf("Pixels per second: %.5f\n", pixelsPerSecondSeq))
	fmt.Print("\nParallel operation\n")
	fmt.Print("---------------------\n")
	fmt.Print(fmt.Sprintf("Time elapsed: %s\n", elaspedTimePara))
	fmt.Print(fmt.Sprintf("Pixels per second: %.5f\n", pixelsPerSecondPara))
	fmt.Print("\n-----------------------------------------------\n")

	// speed calculations
	paraFaster := false
	if pixelsPerSecondSeq < pixelsPerSecondPara {
		paraFaster = true
	}

	var throughputString string
	var executionString string
	if paraFaster {
		throughputIncrease := ((pixelsPerSecondPara - pixelsPerSecondSeq) / pixelsPerSecondSeq) * 100
		executionPercentage := (elaspedInSecondsPara / elaspedInSecondsSeq) * 100
		throughputString = fmt.Sprintf("Parallel algorithm throughput %.5f percent more\n", throughputIncrease)
		executionString = fmt.Sprintf("Parallel agorithm execution time is %.5f percent of sequential execution time\n", executionPercentage)
	} else {
		throughputIncrease := ((pixelsPerSecondSeq - pixelsPerSecondPara) / pixelsPerSecondPara) * 100
		executionPercentage := (elaspedInSecondsSeq / elaspedInSecondsPara) * 100
		throughputString = fmt.Sprintf("Sequential algorithm throughput %.5f percent more\n", throughputIncrease)
		executionString = fmt.Sprintf("Sequential algorithm execution time is %.5f percent of parallel execution time\n", executionPercentage)
	}

	fmt.Print(throughputString)
	fmt.Print(executionString)
}

func demoParaImageProcess(numCpus int, fileName string) {
	runtime.GOMAXPROCS(numCpus)
}

func demoSeqImageProcess(numCpus int, fileName string) {

}

func parallelApplyKernel(row, begin, end int, src *image.NRGBA, dest *image.NRGBA, kernel []float64, wg *sync.WaitGroup) {

	// add routine to the wait group, signal done when completed
	wg.Add(1)
	defer wg.Done()

	for pixel := begin; pixel < end; pixel++ {
		applyKernelPixel(pixel, row, src, dest, kernel)
	}
}

// apply 'kernel' to  pixel at 'x,y' in 'src' put result in 'dest'
func applyKernelPixel(x, y int, src *image.NRGBA, dest *image.NRGBA, kernel []float64) {

	if len(kernel) != 9 {
		panic(1)
	}

	// get the offsets of the pixels used by the 3x3 kernel
	kernelOffsets := getKernelPixelOffsets(x, y, src)
	// if first kernel index in kernelOffsets then we are on an edge pixel
	if kernelOffsets[0] == -1 {
		// so set destination pixel to be same as source pixel
		srcColor := getPixelColorNRGBA(src.PixOffset(x, y), src)
		setPixelColorNRGBA(src.PixOffset(x, y), dest, srcColor)
		return
	}

	colorSum := floatingColor{0, 0, 0}

	// go through each value in the kernel
	for idx, kerVal := range kernel {
		// get the offset of the pixel that corresponds with the current kernel value
		// get the color of said pixel
		currPixColor := getPixelColorNRGBA(kernelOffsets[idx], src)
		// apply the corresponding kernel value to the color
		multRes := multiplyColor(currPixColor, kerVal)
		// sum the colors as we go
		colorSum = addFloatingColor(colorSum, multRes)
	}

	destColor := color.NRGBA{uint8(colorSum.R), uint8(colorSum.G), uint8(colorSum.B), 255}
	dest.Set(x, y, destColor)
}

// get the offsets of the pixels surrounding pixel at location x,y
func getKernelPixelOffsets(x, y int, src *image.NRGBA) []int {

	if x == src.Bounds().Min.X || x == (src.Bounds().Max.X-1) || y == src.Bounds().Min.Y || (y == src.Bounds().Max.Y-1) {
		return []int{-1}
	}

	offsets := make([]int, 9)

	// row 1
	offsets[0] = src.PixOffset(x-1, y-1)
	offsets[1] = src.PixOffset(x, y-1)
	offsets[2] = src.PixOffset(x+1, y-1)

	// row 2
	offsets[3] = src.PixOffset(x-1, y)
	offsets[4] = src.PixOffset(x, y)
	offsets[5] = src.PixOffset(x+1, y)

	// row 3
	offsets[6] = src.PixOffset(x-1, y+1)
	offsets[7] = src.PixOffset(x, y+1)
	offsets[8] = src.PixOffset(x+1, y+1)

	return offsets
}

// get the color of the pixel in 'src' at 'offset'
func getPixelColorNRGBA(offset int, src *image.NRGBA) color.NRGBA {

	// use offset to access pixel and make a color struct
	return color.NRGBA{src.Pix[offset], src.Pix[offset+1], src.Pix[offset+2], src.Pix[offset+3]}
}

// set the color of the pixel in 'src' at 'offset', to 'color'
func setPixelColorNRGBA(offset int, src *image.NRGBA, color color.NRGBA) {
	// use offset to access pixel and change color values
	src.Pix[offset] = color.R   // red
	src.Pix[offset+1] = color.G // green
	src.Pix[offset+2] = color.B // blue
	src.Pix[offset+3] = color.A // alpha
}

// sum the values of the colors in 'slice'
func sumColorSlice(slice []floatingColor) floatingColor {

	var redSum float64
	var greenSum float64
	var blueSum float64

	for i := range slice {
		redSum += slice[i].R
		greenSum += slice[i].G
		blueSum += slice[i].B
	}

	return floatingColor{redSum, greenSum, blueSum}
}

// add two floating color structs togeather
func addFloatingColor(colorA, colorB floatingColor) floatingColor {
	return floatingColor{colorA.R + colorB.R, colorA.G + colorB.G, colorA.B + colorB.B}
}

// multiply the values in 'color' by 'val'
func multiplyColor(origin color.NRGBA, val float64) floatingColor {

	redVal := float64(origin.R) * val
	greenVal := float64(origin.G) * val
	blueVal := float64(origin.B) * val

	return floatingColor{redVal, greenVal, blueVal}
}

// initialise the flags used to operate the program
func initFlags() {
	flag.IntVar(&cpuCountFlag, "cpus", -1, "number of CPU's to use")
	flag.StringVar(&filterFlag, "filter", "", `choose filter type (emboss, leftsobel, outline, bottomsobel, sharpen, edge)`)
	flag.StringVar(&fileName, "file", "", `choose the image`)
	flag.Parse()
}

// parse which filter to use
func setSelectedFilter() {
	switch filterFlag {
	case "emboss":
		selectedFilter = emboss

	case "leftsobel":
		selectedFilter = leftSobel

	case "outline":
		selectedFilter = outline

	case "bottomsobel":
		selectedFilter = bottomSobel

	case "sharpen":
		selectedFilter = sharpen

	case "edge":
		selectedFilter = edge

	default:
		fmt.Print("invalid filter supplied")
		panic(1)
	}

	selectedFilterStr = filterFlag
}

func check(e error) {
	if e != nil {
		fmt.Print(e.Error())
		panic(e)
	}
}

// define some filter kernels
var emboss = []float64{
	-2, -1, -0,
	-1, 1, 1,
	0, 1, 2}

var leftSobel = []float64{
	1, 0, -1,
	2, 0, -2,
	1, 0, -1}

var outline = []float64{
	-1, -1, -1,
	-1, 8, -1,
	-1, -1, -1}

var bottomSobel = []float64{
	-1, -2, -1,
	0, 0, 0,
	1, 2, 1}

var sharpen = []float64{
	0, -1, 0,
	-1, 5, -1,
	0, -1, 0}

var edge = []float64{
	0, 1, 0,
	1, -4, 1,
	0, 1, 0}
