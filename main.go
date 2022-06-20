package main

import "github.com/jackie8tao/ezjob/cmd"

func main() {
	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
