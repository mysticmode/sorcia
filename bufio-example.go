package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func main() {
	file, err := os.Open("gulpfile.js")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var codeLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		codeLines = append(codeLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	for _, eachline := range codeLines {
		fmt.Println(eachline)
	}

	fmt.Println(len(codeLines))
}
