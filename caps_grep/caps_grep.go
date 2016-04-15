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
)

func check(e error) {
	if e != nil {
		//fmt.Print(e.Error())
		panic(e)
	}
}

type empty struct{}
type semaphore chan empty

var typeFlag string
var searchStrFlag string
var verboseFlag bool
var modeFlag string
var progressFlag bool

func main() {

	initFlags()

	var verboseOutput string
	var fullCount int64
	var fileCountMap string

	runtime.GOMAXPROCS(runtime.NumCPU())

	if len(flag.Args()) == 0 {

		fmt.Print("no arguments given to search\n")
		return

	} else if typeFlag == "file" {

		searchFiles(&verboseOutput, &fileCountMap, &fullCount)

	} else if typeFlag == "folder" {

		if modeFlag == "para" {
			searchFolders(&verboseOutput, &fileCountMap, &fullCount)
		} else if modeFlag == "seq" {
			searchFoldersSeq(&verboseOutput, &fileCountMap, &fullCount)
		}

	} else {

		fmt.Print("error incorret type argment given")
	}

	if verboseFlag == true {
		fmt.Print(verboseOutput)
		fmt.Print("\nSearch complete\n")
		fmt.Print("-----------------------------------------------\n")
		fmt.Print(fileCountMap)
		fmt.Print(fmt.Sprintf("Total Occurrences: %d \n", fullCount))
	} else {
		fmt.Print("\nSearch complete\n")
		fmt.Print("-----------------------------------------------\n")
		fmt.Print(fileCountMap)
		fmt.Print(fmt.Sprintf("Total Occurrences: %d \n", fullCount))
	}
}

func jobMaker(fileJobsChan chan<- string, fileList []string, wg *sync.WaitGroup) {
	defer wg.Done()

	for _, fileName := range fileList {
		fileJobsChan <- fileName
	}
	close(fileJobsChan)
}

func worker(targetStr string, fileJob <-chan string, resultsVerbose, resultsFileCountMap chan<- string, totalCount chan<- int64, wg *sync.WaitGroup, id int, openedFilesSem semaphore) {

	defer wg.Done()

	originFileName := <-fileJob

	// acquire resource for opening files
	x := empty{}
	openedFilesSem <- x
	fileData, err := ioutil.ReadFile(originFileName)
	if err == nil {
		// if name was qualified chop it down to the base / showten if need be
		fileName := filepath.Base(originFileName)
		if len(fileName) > 25 {
			fileName = fileName[:25] + "..."
		}

		// search the given file for the target string then put results into channels
		if true == progressFlag {
			fmt.Print(fmt.Sprintf("Searching file: %s \n", fileName))
		}
		output, numFound := searchBytes(targetStr, fileData, fileName)
		totalCount <- numFound
		resultsVerbose <- output
		resultsFileCountMap <- fmt.Sprintf("%s : %d occurrences\n", fileName, numFound)
	} else {
		//check(err)
		fmt.Print(err.Error() + "\n")
	}

	// release resource for opening files
	<-openedFilesSem
}

func fileCountCollector(fullCount *int64, occurrenceCountChan <-chan int64, items int64, wg *sync.WaitGroup) {
	defer wg.Done()
	var sumOccurences int64

	for i := int64(0); i < items; i++ {
		sumOccurences += <-occurrenceCountChan
	}

	*fullCount = sumOccurences
}

func verboseOutputCollector(verboseOutput *string, verboseOutputChan <-chan string, items int64, wg *sync.WaitGroup) {
	defer wg.Done()
	var verboseOutputBuffer bytes.Buffer

	for i := int64(0); i < items; i++ {
		verboseOutputBuffer.WriteString(<-verboseOutputChan)
	}

	*verboseOutput = verboseOutputBuffer.String()
}

func fileCountMapCollector(fileCountMap *string, fileCountMapChan <-chan string, items int64, wg *sync.WaitGroup) {
	defer wg.Done()
	var fileCountMapBuffer bytes.Buffer

	for i := int64(0); i < items; i++ {
		fileCountMapBuffer.WriteString(<-fileCountMapChan)
	}

	*fileCountMap = fileCountMapBuffer.String()
}

func searchFolders(verboseOutput, fileCountMap *string, fullCount *int64) {

	fileList := []string{}

	// build up the list of files that we are going to search
	for _, directoryArg := range flag.Args() {
		// walk through each directory and get the names of files
		err := filepath.Walk(directoryArg, func(path string, fileInfo os.FileInfo, err error) error {
			// check that we are only listing directories and .txt files in our list
			if false == fileInfo.IsDir() {
				fileList = append(fileList, path)
			}
			return nil
		})
		check(err)
	}

	var wg sync.WaitGroup
	fileListLen := int64(len(fileList))
	openedFiles := make(semaphore, 100)
	fileJobsChan := make(chan string, 10)
	verboseOutputChan := make(chan string, fileListLen)
	fileCountMapChan := make(chan string, fileListLen)
	occurrenceCountChan := make(chan int64, fileListLen)

	// kick off all the workers who wait to be given jobs
	for i := range fileList {
		wg.Add(1)
		go worker(searchStrFlag, fileJobsChan, verboseOutputChan, fileCountMapChan, occurrenceCountChan, &wg, i, openedFiles)
	}

	// create all the jobs
	wg.Add(1)
	go jobMaker(fileJobsChan, fileList, &wg)

	// collect the results
	wg.Add(3)
	go fileCountCollector(fullCount, occurrenceCountChan, fileListLen, &wg)
	go fileCountMapCollector(fileCountMap, fileCountMapChan, fileListLen, &wg)
	go verboseOutputCollector(verboseOutput, verboseOutputChan, fileListLen, &wg)

	wg.Wait()
}

func searchFiles(verboseOutput, fileCountMap *string, fullCount *int64) {

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

	*verboseOutput = resultsBuffer.String()
	*fileCountMap = fileReportBuffer.String()
	*fullCount = totalOccurrences
}

func searchFoldersSeq(fullResult, fileCountMap *string, fullCount *int64) {

	fileList := []string{}

	var resultsBuffer bytes.Buffer
	var totalOccurrences int64
	var fileReportBuffer bytes.Buffer

	// build up the list of files that we are going to search
	for _, directoryArg := range flag.Args() {
		// walk through each directory and get the names of files
		err := filepath.Walk(directoryArg, func(path string, fileInfo os.FileInfo, err error) error {
			// check that we are only listing directories and .txt files in our list
			if false == fileInfo.IsDir() {
				fileList = append(fileList, path)
			}
			return nil
		})
		check(err)
	}

	// go through the file list and search each file
	for _, originFileName := range fileList {
		// read the file into a byte buffer
		fileData, err := ioutil.ReadFile(originFileName)
		if err == nil {
			// if name was qualified chop it down to the base / showten if need be
			fileName := filepath.Base(originFileName)
			if len(fileName) > 25 {
				fileName = fileName[:25] + "..."
			}

			// search the file for the target string
			if true == progressFlag {
				fmt.Print(fmt.Sprintf("Searching file: %s \n", fileName))
			}
			output, numFound := searchBytes(searchStrFlag, []byte(fileData), fileName)

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

func initFlags() {

	// setup the type flag
	flag.StringVar(&typeFlag, "type", "file", `specify if either file names or folders names will be provided to search`)
	flag.StringVar(&searchStrFlag, "str", "null", `the string to search for`)
	flag.BoolVar(&verboseFlag, "verbose", false, `if flase only shows the number of occurrences, if true shows locations too`)
	flag.StringVar(&modeFlag, "mode", "seq", `search in either sequential of parrallel mode`)
	flag.BoolVar(&progressFlag, "progress", true, `show the file currently being searched `)

	flag.Parse()
}

// search the for the targetStr in the given byte array "data"
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
