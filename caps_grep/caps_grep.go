package main

import (
	//"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {

	searchStr := os.Args[1]
	fileData, err := ioutil.ReadFile(os.Args[2])
	check(err)

	fmt.Print(searchBytes(searchStr, fileData, os.Args[2]))

}

func searchBytes(targetStr string, data []byte, fileName string) string {

	var outputBuffer bytes.Buffer
	targetMatches := len(targetStr)
	var lastChar byte = ' '

	outputBuffer.WriteString(fmt.Sprintf("File: %s \n", fileName))

	// information about position in the current file
	var lineNum, charNum, lastLineEnd int

	if targetMatches > 0 {
		lineNum = 1
	} else {
		return ""
	}

	// infomation about the search process
	var occurrencesFound, matchedChars int

	for byteIdx, byteVal := range data {

		switch byteVal {

		case '\n', '\r':
			lineNum++
			lastLineEnd = byteIdx
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
			outputBuffer.WriteString(fmt.Sprintf("Match @ line: %d, pos: %d, %s \n", lineNum, pos, line))
		}

		lastChar = byteVal
		charNum++
	}

	outputBuffer.WriteString(fmt.Sprintf("Total occurrences: %d \n", occurrencesFound))
	return outputBuffer.String()
}
