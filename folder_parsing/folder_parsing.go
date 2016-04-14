package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	searchDir := os.Args[1]

	fileList := []string{}
	err := filepath.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
		fileList = append(fileList, path)
		return nil
	})
	check(err)

	for _, file := range fileList {
		fmt.Println(file)
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
