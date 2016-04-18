package main

import (
	"fmt"
	"github.com/disintegration/imaging"
	"image"
	"image/color"
	"os"
)

type floatingColor struct {
	R float64
	G float64
	B float64
	A float64
}

func check(e error) {
	if e != nil {
		fmt.Print(e.Error())
		panic(e)
	}
}

func main() {

	// open the image
	srcImg, err := imaging.Open(os.Args[1])
	check(err)

	// create a struct representation of the image
	srcImgNRGB := imaging.Clone(srcImg)

	// crate a struct to put the new ouput image into
	outWidth := srcImg.Bounds().Max.X
	outHeight := srcImg.Bounds().Max.Y
	outImage := image.NewNRGBA(image.Rect(0, 0, outWidth, outHeight))

	// iterate through each pixel stored in the NRGBA struct
	for rowY := 0; rowY < outHeight; rowY++ {
		for pixelX := 0; pixelX < outWidth; pixelX++ {

			pixOffset := srcImgNRGB.PixOffset(pixelX, rowY)
			srcColor := getPixelColorNRGBA(pixOffset, srcImgNRGB)

			resA := 0.299*float64(srcColor.R) + 0.587*float64(srcColor.G) + 0.114*float64(srcColor.B)
			colorRes := uint8(resA + 0.5)
			destColor := color.NRGBA{colorRes, colorRes, colorRes, srcColor.A}

			setPixelColorNRGBA(pixOffset, outImage, destColor)
		}
	}

	//outputName := fmt.Sprintf("%s_grey.jpg", os.Args[1])
	err = imaging.Save(outImage, "outputName.jpg")
	if err != nil {
		panic(err)
	}
}

func getKernelPixelOffsets(x, y int, src *image.NRGBA) []int {

	if x == 0 || x == src.Bounds().Max.X || y == 0 || y == src.Bounds().Max.Y {
		offsets := make([]int, 9)
		memset(offsets, -1)
		return offsets
	}

	offsets := make([]int, 9)

	// row 1
	offsets[0] = src.PixOffset(x-1, y-1)
	offsets[1] = src.PixOffset(x, y-1)
	offsets[2] = src.PixOffset(x+1, y)

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

func getPixelColorNRGBA(offset int, src *image.NRGBA) color.NRGBA {

	// use offset to access pixel and make a color struct
	return color.NRGBA{src.Pix[offset], src.Pix[offset+1], src.Pix[offset+2], src.Pix[offset+3]}
}

func setPixelColorNRGBA(offset int, src *image.NRGBA, color color.NRGBA) {

	// use offset to access pixel and change color values
	src.Pix[offset] = color.R   // red
	src.Pix[offset+1] = color.G // green
	src.Pix[offset+2] = color.B // blue
	src.Pix[offset+3] = color.A // alpha
}

func sumColorSlice(slice []floatingColor) floatingColor {

	var redSum float64
	var greenSum float64
	var blueSum float64
	var alphaSum float64

	for i := range slice {
		redSum += slice[i].R
		greenSum += slice[i].G
		blueSum += slice[i].B
		alphaSum += slice[i].A
	}
}

func multiplyColor(origin color.NRGBA, val float64) floatingColor {

	redVal := float64(origin.R) * val
	greenVal := float64(origin.G) * val
	blueVal := float64(origin.B) * val
	alphaVal := float64(origin.A) * val

	return floatingColor{redVal, greenVal, blueVal, alphaVal}
}

func memset(a []int, v int) {
	for i := range a {
		a[i] = v
	}
}
