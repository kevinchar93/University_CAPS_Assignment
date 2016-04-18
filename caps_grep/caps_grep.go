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

var typeFlag string
var searchStrFlag string
var verboseFlag bool
var methodFLag string
var progressFlag bool
var fileMapFlag bool

var cpuCount int
var fileName string
var cpuCountFlag int

func main() {

	initFlags()

	// get the ammount of CPU's to use
	if cpuCountFlag == 0 {
		cpuCount = 1
	} else if cpuCountFlag > runtime.NumCPU() || cpuCountFlag == -1 {
		cpuCount = runtime.NumCPU()
	} else {
		cpuCount = cpuCountFlag
	}
	fmt.Print(fmt.Sprintf("Using %d cpus\n", cpuCount))
	runtime.GOMAXPROCS(cpuCount)

	if len(flag.Args()) == 0 {

		fmt.Print("no arguments given to search\n")
		return
	}

	// sequential operation ----------------------------------------------------
	var verboseOutputSeq string
	var fullCountSeq int64
	var charCountSeq int64
	var fileCountSeq int64
	var fileCountMapSeq string

	fmt.Print("\nBegin sequential\n")
	startTimeSeq := time.Now()
	searchFoldersSeq(&verboseOutputSeq, &fileCountMapSeq, &fileCountSeq, &charCountSeq, &fullCountSeq)
	elaspedSeq := time.Since(startTimeSeq)
	fmt.Print("End sequential\n")
	elaspedInSecondsSeq := elaspedSeq.Seconds()
	charsPerSecondSeq := float64(charCountSeq) / elaspedInSecondsSeq
	filesPerSecondSeq := float64(fileCountSeq) / elaspedInSecondsSeq

	//--------------------------------------------------------------------------

	// parallel operation ------------------------------------------------------
	var verboseOutputPara string
	var fullCountPara int64
	var charCountPara int64
	var fileCountPara int64
	var fileCountMapPara string

	fmt.Print("Begin parallel\n")
	startTimePara := time.Now()
	searchFoldersPara(&verboseOutputPara, &fileCountMapPara, &fileCountPara, &charCountPara, &fullCountPara)
	elaspedPara := time.Since(startTimePara)
	fmt.Print("End parallel\n")
	elaspedInSecondsPara := elaspedPara.Seconds()
	charsPerSecondPara := float64(charCountPara) / elaspedInSecondsPara
	filesPerSecondPara := float64(fileCountPara) / elaspedInSecondsPara
	//--------------------------------------------------------------------------

	// operation report --------------------------------------------------------
	fmt.Print("\nSummary\n")
	fmt.Print("-----------------------------------------------\n")
	fmt.Print(fmt.Sprintf("CPUs : %d", cpuCount))
	fmt.Print("\nSequential operation\n")
	fmt.Print("---------------------\n")
	fmt.Print(fmt.Sprintf("Search string: \"%s\" Total occurrences: %d \n", searchStrFlag, fullCountSeq))
	fmt.Print(fmt.Sprintf("Characters scanned: %d \n", charCountSeq))
	fmt.Print(fmt.Sprintf("Files scanned: %d \n", fileCountSeq))
	fmt.Print(fmt.Sprintf("Time elasped: %s \n", elaspedSeq))
	fmt.Print(fmt.Sprintf("Characters per second: %.5f \n", charsPerSecondSeq))
	fmt.Print(fmt.Sprintf("Files per second: %.5f \n", filesPerSecondSeq))
	fmt.Print("\nParallel operation\n")
	fmt.Print("---------------------\n")
	fmt.Print(fmt.Sprintf("Search string: \"%s\" Total occurrences: %d \n", searchStrFlag, fullCountPara))
	fmt.Print(fmt.Sprintf("Characters scanned: %d \n", charCountPara))
	fmt.Print(fmt.Sprintf("Files scanned: %d \n", fileCountPara))
	fmt.Print(fmt.Sprintf("Time elasped: %s \n", elaspedPara))
	fmt.Print(fmt.Sprintf("Characters per second: %.5f \n", charsPerSecondPara))
	fmt.Print(fmt.Sprintf("Files per second: %.5f \n", filesPerSecondPara))
	fmt.Print("\n-----------------------------------------------\n")

	// speed calculations
	paraFaster := false
	if elaspedInSecondsPara < elaspedInSecondsSeq {
		paraFaster = true
	}

	var fileThroughputString string
	var charThroughputString string
	var executionString string

	if paraFaster {
		fileThroughputIncrease := ((filesPerSecondPara - filesPerSecondSeq) / filesPerSecondSeq) * 100
		charThroughputIncrease := ((charsPerSecondPara - charsPerSecondSeq) / charsPerSecondSeq) * 100
		executionPercentage := (elaspedInSecondsPara / elaspedInSecondsSeq) * 100
		fileThroughputString = fmt.Sprintf("Parallel algorithm file throughput %.5f percent more\n", fileThroughputIncrease)
		charThroughputString = fmt.Sprintf("Parallel algorithm character throughput %.5f percent more\n", charThroughputIncrease)
		executionString = fmt.Sprintf("Parallel agorithm execution time is %.5f percent of sequential execution time\n", executionPercentage)
	} else {
		fileThroughputIncrease := ((filesPerSecondSeq - filesPerSecondPara) / filesPerSecondPara) * 100
		charThroughputIncrease := ((charsPerSecondSeq - charsPerSecondPara) / charsPerSecondPara) * 100
		executionPercentage := (elaspedInSecondsSeq / elaspedInSecondsPara) * 100
		fileThroughputString = fmt.Sprintf("Sequential algorithm file throughput %.5f percent more\n", fileThroughputIncrease)
		charThroughputString = fmt.Sprintf("Sequential algorithm character throughput %.5f percent more\n", charThroughputIncrease)
		executionString = fmt.Sprintf("Sequential agorithm execution time is %.5f percent of parallel execution time\n", executionPercentage)
	}

	fmt.Print(fileThroughputString)
	fmt.Print(charThroughputString)
	fmt.Print(executionString)
}

// search the folders provided in the arguments to the program - search is done in parallel
func searchFoldersPara(verboseOutput, fileCountMap *string, fileCount, charCount, fullCount *int64) {

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
		wg.Add(1)
		go worker(searchStrFlag, fileJobsChan, verboseOutputChan, fileCountMapChan, charCountChan, occurrenceCountChan, &wg, i, openedFiles)
	}

	// create all the job
	wg.Add(1)
	go jobMaker(fileJobsChan, fileList, &wg)

	// collect the results
	wg.Add(4)
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

// routines that "makes" jobs (filenames) and puts them in a channel for workes to receive
func jobMaker(fileJobsChan chan<- string, fileList []string, wg *sync.WaitGroup) {
	defer wg.Done()

	for _, fileName := range fileList {
		fileJobsChan <- fileName
	}
}

// routine that performs the actual searching task
func worker(targetStr string, fileJob <-chan string, resultsVerboseChan, resultsFileCountMapChan chan<- string,
	charCountChan, totalCountChan chan<- int64, wg *sync.WaitGroup, id int, openedFilesSem semaphore) {

	defer wg.Done()

	originFileName := <-fileJob

	// acquire resource for opening files
	sem := empty{}
	openedFilesSem <- sem

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
	defer wg.Done()
	var sumOccurences int64

	for i := int64(0); i < items; i++ {
		sumOccurences += <-occurrenceCountChan
	}

	*fullCount = sumOccurences
}

// routine to collect the verbose output from the verboseOutputChan and put it into the string verboseOutput
func verboseOutputCollector(verboseOutput *string, verboseOutputChan <-chan string, items int64, wg *sync.WaitGroup) {
	defer wg.Done()
	var verboseOutputBuffer bytes.Buffer

	for i := int64(0); i < items; i++ {
		verboseOutputBuffer.WriteString(<-verboseOutputChan)
	}

	*verboseOutput = verboseOutputBuffer.String()
}

// routine to collect the file names with thier occurrence counts from fileCountMapChan and put it into fileCountMap
func fileCountMapCollector(fileCountMap *string, fileCountMapChan <-chan string, items int64, wg *sync.WaitGroup) {
	defer wg.Done()
	var fileCountMapBuffer bytes.Buffer

	for i := int64(0); i < items; i++ {
		fileCountMapBuffer.WriteString(<-fileCountMapChan)
	}

	*fileCountMap = fileCountMapBuffer.String()
}

// routine to collect the number of characters from charCountChan and put it into charCount
func charCountCollector(charCount *int64, charCountChan <-chan int64, items int64, wg *sync.WaitGroup) {
	defer wg.Done()
	var sumChars int64

	for i := int64(0); i < items; i++ {
		sumChars += <-charCountChan
	}

	*charCount = sumChars
}

// setup the flag arguments that the program uses
func initFlags() {

	flag.StringVar(&typeFlag, "type", "folder", `specify if either file names or folders names will be provided to search`)
	flag.StringVar(&searchStrFlag, "str", "null", `the string to search for`)
	flag.BoolVar(&verboseFlag, "verbose", false, `if flase only shows the number of occurrences, if true shows locations too`)
	flag.StringVar(&methodFLag, "method", "seq", `search in using either the sequential or parrallel method`)
	flag.BoolVar(&progressFlag, "progress", false, `show the file currently being searched `)
	flag.BoolVar(&fileMapFlag, "fileMap", false, `show how many occurences each file had`)
	flag.IntVar(&cpuCountFlag, "cpus", -1, "number of CPU's to use")

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
