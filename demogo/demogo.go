package main

import (
	"flag"
	"fmt"
	"github.com/disintegration/imaging"
	"image"
	"image/color"
)

type floatingColor struct {
	R float64
	G float64
	B float64
}

var filterFlag string
var selectedFilter []float64
var selectedFilterStr string
var fileName string

func main() {

	initFlags()
	setSelectedFilter()

	// open the image
	srcImg, err := imaging.Open(fileName)
	check(err)

	// create a struct representation of the image
	srcImgNRGB := imaging.Clone(srcImg)

	// get the indexs to begin and end at
	beginX := srcImg.Bounds().Min.X
	beginY := srcImg.Bounds().Min.Y

	// crate a struct to put the new ouput image into
	outWidth := srcImg.Bounds().Max.X
	outHeight := srcImg.Bounds().Max.Y
	outImage := image.NewNRGBA(image.Rect(0, 0, outWidth, outHeight))

	// iterate through each pixel stored in the NRGBA struct
	for rowY := beginY; rowY < outHeight; rowY++ {
		for pixelX := beginX; pixelX < outWidth; pixelX++ {
			applyKernelPixel(pixelX, rowY, srcImgNRGB, outImage, selectedFilter)
		}
	}

	outputName := fmt.Sprintf("%s_%s_output.jpg", fileName, selectedFilterStr)
	err = imaging.Save(outImage, outputName)
	if err != nil {
		panic(err)
	}
}

// apply 'kernel' to pixel at 'x,y' in 'src' put result in 'dest'
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

	// temp storage for all the intermediate results created by the kernel
	colorSumSlice := make([]floatingColor, 9)

	// go through each value in the kernel
	for idx, kerVal := range kernel {
		// get the offset of the pixel that corresponds with the current kernel value
		currPixOffset := kernelOffsets[idx]
		// get the color of said pixel
		currPixColor := getPixelColorNRGBA(currPixOffset, src)
		// apply the corresponding kernel value to the color
		multRes := multiplyColor(currPixColor, kerVal)
		// place the resultant color in the temp slice
		colorSumSlice[idx] = multRes
	}

	cSum := sumColorSlice(colorSumSlice)
	destColor := color.NRGBA{uint8(cSum.R), uint8(cSum.G), uint8(cSum.B), 255}

	// get the off set of the pixel we wish to change
	pixelOffset := src.PixOffset(x, y)
	setPixelColorNRGBA(pixelOffset, dest, destColor)
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

// multiply the values in 'color' by 'val'
func multiplyColor(origin color.NRGBA, val float64) floatingColor {

	redVal := float64(origin.R) * val
	greenVal := float64(origin.G) * val
	blueVal := float64(origin.B) * val

	return floatingColor{redVal, greenVal, blueVal}
}

// initialise the flags used to operate the program
func initFlags() {
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
