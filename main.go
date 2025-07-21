package main

import (
	"fmt"
)

func main(){
	fmt.Println("Input:")
	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		fmt.Println("Error reading input")
	}
	fmt.Println("input")
}
