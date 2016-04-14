package main

import (
	//"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	//"os"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

var typeFlag string
var searchStrFlag string
var verboseFlag bool

func main() {

	initFlags()

	var fullResult string
	var fullCount int64
	var fileCountMap string

	if typeFlag == "file" {

		searchFiles(&fullResult, &fileCountMap, &fullCount)

	} else if typeFlag == "folder" {
		// go thorugh the folder recursively and get a list of file names
		// search each of the file names once we have them
		fmt.Print("search the given folder for search term")
	} else {
		fmt.Print("error incorret type argment given")
	}

	if verboseFlag == true {
		fmt.Print(fullResult)
		fmt.Print("Search complete\n")
		fmt.Print("-----------------------------------------------\n")
		fmt.Print(fileCountMap)
		fmt.Print(fmt.Sprintf("Total Occurrences: %d \n", fullCount))
	} else {
		fmt.Print("Search complete\n")
		fmt.Print("-----------------------------------------------\n")
		fmt.Print(fileCountMap)
		fmt.Print(fmt.Sprintf("Total Occurrences: %d \n", fullCount))
	}
}

func searchFiles(fullResult, fileCountMap *string, fullCount *int64) {

	var resultsBuffer bytes.Buffer
	var totalOccurrences int64
	var fileReportBuffer bytes.Buffer

	// do this for each of the supplied arguments
	for _, originFileName := range flag.Args() {

		// read the file into a byte buffer
		fileData, err := ioutil.ReadFile(originFileName)
		if err == nil {
			// if name was qualified chop it down to the base / showten if need be
			fileName := filepath.Base(originFileName)
			if len(fileName) > 25 {
				fileName = fileName[:25] + "..."
			}

			// search the file for the target string
			fmt.Print(fmt.Sprintf("Searching file: %s \n", fileName))
			output, numFound := searchBytes(searchStrFlag, fileData, fileName)

			// update variables storing information
			totalOccurrences += numFound
			resultsBuffer.WriteString(output)
			fileReportBuffer.WriteString(fmt.Sprintf("%s : %d occurrences\n", fileName, numFound))
		} else {
			check(err)
		}
	}

	*fullResult = resultsBuffer.String()
	*fileCountMap = fileReportBuffer.String()
	*fullCount = totalOccurrences
}

func searchFolders(fullResult, fileCountMap *string, fullCount *int64) {

}

func initFlags() {

	// setup the type flag
	flag.StringVar(&typeFlag, "type", "file", `specify if either file names or folders names will be provided to search`)
	flag.StringVar(&searchStrFlag, "str", "null", `the string to search for`)
	flag.BoolVar(&verboseFlag, "verbose", true, `if flase only shows the number of occurrences, if true shows locations too`)

	flag.Parse()
}

// search the data in
func searchBytes(targetStr string, data []byte, fileName string) (string, int64) {

	var outputBuffer bytes.Buffer
	targetMatches := int64(len(targetStr))
	var lastChar byte = ' '

	//outputBuffer.WriteString(fmt.Sprintf("File: %s \n", fileName))

	// information about position in the current file
	var lineNum, charNum, lastLineEnd int64

	if targetMatches > 0 {
		lineNum = 1
	} else {
		return "", 0
	}

	// infomation about the search process
	var occurrencesFound, matchedChars int64

	for byteIdx, byteVal := range data {

		switch byteVal {

		case '\n', '\r':
			lineNum++
			lastLineEnd = int64(byteIdx)
			matchedChars = 0
			charNum = 0

		default:
			if matchedChars == 0 {
				if targetStr[matchedChars] == byteVal && lastChar == ' ' {
					matchedChars++
				}
			} else if targetStr[matchedChars] == byteVal {
				// another matching byte was found
				matchedChars++
			} else {
				// the bytes did not match
				matchedChars = 0
			}
		}

		// a match was found
		if targetMatches == matchedChars {
			matchedChars = 0
			occurrencesFound++
			pos := charNum - (targetMatches - 1)
			line := string(data[lastLineEnd+1 : lastLineEnd+charNum+1])
			outputBuffer.WriteString(fmt.Sprintf("Match in :%s line: %d, pos: %d, %s \n", fileName, lineNum, pos, line))
		}

		lastChar = byteVal
		charNum++
	}

	//outputBuffer.WriteString(fmt.Sprintf("Total occurrences: %d \n", occurrencesFound))
	return outputBuffer.String(), occurrencesFound
}
