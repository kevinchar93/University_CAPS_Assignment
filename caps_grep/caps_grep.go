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
var fileMapFlag bool
var cpuCount int

func main() {

	initFlags()

	var verboseOutput string
	var fullCount int64
	var charCount int64
	var fileCount int64
	var fileCountMap string

	cpuCount = runtime.NumCPU()
	runtime.GOMAXPROCS(cpuCount)

	startTime := time.Now()

	if len(flag.Args()) == 0 {

		fmt.Print("no arguments given to search\n")
		return

	} else if typeFlag == "file" {

		searchFiles(&verboseOutput, &fileCountMap, &fileCount, &charCount, &fullCount)

	} else if typeFlag == "folder" {

		if modeFlag == "para" {
			searchFolders(&verboseOutput, &fileCountMap, &fileCount, &charCount, &fullCount)
		} else if modeFlag == "seq" {
			searchFoldersSeq(&verboseOutput, &fileCountMap, &fileCount, &charCount, &fullCount)
		}
	} else {

		fmt.Print("error incorret type argment given")
		return
	}

	elasped := time.Since(startTime)
	elaspedInSeconds := elasped.Seconds()
	charsPerSecond := float64(charCount) / elaspedInSeconds
	filesPerSecond := float64(fileCount) / elaspedInSeconds

	// print results
	fmt.Print("\nSearch complete\n")
	fmt.Print("-----------------------------------------------\n")

	if verboseFlag {
		fmt.Print(verboseOutput)
	}

	if fileMapFlag {
		fmt.Print(fileCountMap)
	}

	fmt.Print("\nSummary\n")
	fmt.Print("-----------------------------------------------\n")
	fmt.Print(fmt.Sprintf("Search string: \"%s\" Total occurrences: %d \n", searchStrFlag, fullCount))
	fmt.Print(fmt.Sprintf("Time elasped: %s \n", elasped))
	fmt.Print(fmt.Sprintf("Characters scanned: %d \n", charCount))
	fmt.Print(fmt.Sprintf("Characters per second: %.5f \n", charsPerSecond))
	fmt.Print(fmt.Sprintf("Files scanned: %d \n", fileCount))
	fmt.Print(fmt.Sprintf("Files per second: %.5f \n", filesPerSecond))
}

// routines that "makes" jobs (filenames) and puts them in a channel for workes to receive
func jobMaker(fileJobsChan chan<- string, fileList []string, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	for _, fileName := range fileList {
		fileJobsChan <- fileName
	}
	close(fileJobsChan)
}

// routine that performs the actual searching task
func worker(targetStr string, fileJob <-chan string, resultsVerboseChan, resultsFileCountMapChan chan<- string,
	charCountChan, totalCountChan chan<- int64, wg *sync.WaitGroup, id int, openedFilesSem semaphore) {

	wg.Add(1)
	defer wg.Done()

	originFileName := <-fileJob

	// acquire resource for opening files
	x := empty{}
	openedFilesSem <- x

	// open and read from file
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
		output, numFound, numChars := searchBytes(targetStr, fileData, fileName)
		totalCountChan <- numFound
		charCountChan <- numChars
		resultsVerboseChan <- output
		resultsFileCountMapChan <- fmt.Sprintf("%s : %d occurrences\n", fileName, numFound)
	} else {
		//check(err)
		fmt.Print(err.Error() + "\n")
	}

	// release resource for opening files
	<-openedFilesSem
}

// routine to collect the number of matches from occurrenceCountChan and put it into fullCount
func occurrenceCountCollector(fullCount *int64, occurrenceCountChan <-chan int64, items int64, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	var sumOccurences int64

	for i := int64(0); i < items; i++ {
		sumOccurences += <-occurrenceCountChan
	}

	*fullCount = sumOccurences
}

// routine to collect the verbose output from the verboseOutputChan and put it into the string verboseOutput
func verboseOutputCollector(verboseOutput *string, verboseOutputChan <-chan string, items int64, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	var verboseOutputBuffer bytes.Buffer

	for i := int64(0); i < items; i++ {
		verboseOutputBuffer.WriteString(<-verboseOutputChan)
	}

	*verboseOutput = verboseOutputBuffer.String()
}

// routine to collect the file names with thier occurrence counts from fileCountMapChan and put it into fileCountMap
func fileCountMapCollector(fileCountMap *string, fileCountMapChan <-chan string, items int64, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	var fileCountMapBuffer bytes.Buffer

	for i := int64(0); i < items; i++ {
		fileCountMapBuffer.WriteString(<-fileCountMapChan)
	}

	*fileCountMap = fileCountMapBuffer.String()
}

// routine to collect the number of characters from charCountChan and put it into charCount
func charCountCollector(charCount *int64, charCountChan <-chan int64, items int64, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	var sumChars int64

	for i := int64(0); i < items; i++ {
		sumChars += <-charCountChan
	}

	*charCount = sumChars
}

// search the folders provided in the arguments to the program - search is done in parallel
func searchFolders(verboseOutput, fileCountMap *string, fileCount, charCount, fullCount *int64) {

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

	*fileCount = int64(len(fileList))

	var wg sync.WaitGroup
	fileListLen := int64(len(fileList))
	openedFiles := make(semaphore, cpuCount)
	fileJobsChan := make(chan string, cpuCount)
	verboseOutputChan := make(chan string, cpuCount)
	fileCountMapChan := make(chan string, cpuCount)
	occurrenceCountChan := make(chan int64, cpuCount)
	charCountChan := make(chan int64, cpuCount)

	// kick off all the workers who wait to be given jobs
	for i := range fileList {
		go worker(searchStrFlag, fileJobsChan, verboseOutputChan, fileCountMapChan, charCountChan, occurrenceCountChan, &wg, i, openedFiles)
	}

	// create all the jobs
	go jobMaker(fileJobsChan, fileList, &wg)

	// collect the results
	go occurrenceCountCollector(fullCount, occurrenceCountChan, fileListLen, &wg)
	go fileCountMapCollector(fileCountMap, fileCountMapChan, fileListLen, &wg)
	go verboseOutputCollector(verboseOutput, verboseOutputChan, fileListLen, &wg)
	go charCountCollector(charCount, charCountChan, fileListLen, &wg)

	wg.Wait()

	close(verboseOutputChan)
	close(fileCountMapChan)
	close(occurrenceCountChan)
	close(charCountChan)
}

// search the files provided in the arguments to the program - search is done sequentially
func searchFiles(verboseOutput, fileCountMap *string, fileCount, charCount, fullCount *int64) {

	var resultsBuffer bytes.Buffer
	var totalOccurrences int64
	var totalChars int64
	var fileReportBuffer bytes.Buffer

	// do this for each of the supplied arguments
	for i, originFileName := range flag.Args() {

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
			output, numFound, numChars := searchBytes(searchStrFlag, fileData, fileName)

			// update variables storing information
			totalOccurrences += numFound
			totalChars += numChars
			resultsBuffer.WriteString(output)
			fileReportBuffer.WriteString(fmt.Sprintf("%s : %d occurrences\n", fileName, numFound))
		} else {
			check(err)
		}
		*fileCount = int64(i)
	}

	*verboseOutput = resultsBuffer.String()
	*fileCountMap = fileReportBuffer.String()
	*fullCount = totalOccurrences
	*charCount = totalChars
}

// search the folders provided in the arguments to the program - search is done sequentially
func searchFoldersSeq(fullResult, fileCountMap *string, fileCount, charCount, fullCount *int64) {

	fileList := []string{}

	var resultsBuffer bytes.Buffer
	var totalOccurrences int64
	var totalChars int64
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
			output, numFound, numChars := searchBytes(searchStrFlag, []byte(fileData), fileName)

			// update variables storing information
			totalOccurrences += numFound
			totalChars += numChars
			resultsBuffer.WriteString(output)
			fileReportBuffer.WriteString(fmt.Sprintf("%s : %d occurrences\n", fileName, numFound))
		} else {
			check(err)
		}
	}

	*fileCount = int64(len(fileList))
	*fullResult = resultsBuffer.String()
	*fileCountMap = fileReportBuffer.String()
	*fullCount = totalOccurrences
	*charCount = totalChars
}

// setup the flag arguments that the program uses
func initFlags() {

	flag.StringVar(&typeFlag, "type", "folder", `specify if either file names or folders names will be provided to search`)
	flag.StringVar(&searchStrFlag, "str", "null", `the string to search for`)
	flag.BoolVar(&verboseFlag, "verbose", false, `if flase only shows the number of occurrences, if true shows locations too`)
	flag.StringVar(&modeFlag, "mode", "seq", `search in either sequential of parrallel mode`)
	flag.BoolVar(&progressFlag, "progress", false, `show the file currently being searched `)
	flag.BoolVar(&fileMapFlag, "fileMap", true, `show how many occurences each file had`)

	flag.Parse()
}

// search the for the targetStr in the given byte array "data"
func searchBytes(targetStr string, data []byte, fileName string) (string, int64, int64) {

	var outputBuffer bytes.Buffer
	targetMatches := int64(len(targetStr))
	var lastChar byte = ' '
	var charCount int64

	//outputBuffer.WriteString(fmt.Sprintf("File: %s \n", fileName))

	// information about position in the current file
	var lineNum, charNum, lastLineEnd int64

	if targetMatches > 0 {
		lineNum = 1
	} else {
		return "", 0, 0
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
		charCount++
	}

	//outputBuffer.WriteString(fmt.Sprintf("Total occurrences: %d \n", occurrencesFound))
	return outputBuffer.String(), occurrencesFound, charCount
}
