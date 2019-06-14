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

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fmt.Println("<p>" + scanner.Text() + "</p>")
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
