package main

import (
	"fmt"
	"os"
)

var argMap = make(map[string]funcc)

func init() {
	set()
}

func set() {
	argMap["init"] = func() {
		currentDir, err := os.Getwd()
		if err != nil {
			fmt.Println("current working directory cannot be found ", err)
			os.Exit(1)
		}

		newFolder := fmt.Sprintf("%s/.helper", currentDir)

		err = os.Mkdir(newFolder, 0755)
		if err != nil {
			fmt.Println("cannot make directory ", err)
			os.Exit(1)
		}

		fmt.Println("Successfully initialized...")
	}
}

type funcc func()

func main() {

	args := getInput()
	if len(args) < 2 {
		fmt.Println("please provide the arguments")
		os.Exit(1)
	}

	result := argMap[args[1]]
	if result == nil {
		fmt.Println("please provide a valid argument")
		os.Exit(1)
	}

	result()

}

func getInput() []string {
	return os.Args
}
