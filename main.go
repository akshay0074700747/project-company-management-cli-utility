package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type funcc func()

// stores all the tags and the tasks associated with the tag
var (
	initDir     = ".manager"
	argMap      = make(map[string]funcc)
	snapshotKey string
)

// for setting mapping the tag with its associated functionality
func set() {

	//for creating an empty repository which will hold the snapshots
	argMap["init"] = func() {
		// for getting the absolute path of the current working directory
		currentDir, err := os.Getwd()
		if err != nil {
			fmt.Println("current working directory cannot be found ", err)
			os.Exit(1)
		}

		// appends the name of the folder to be created with the absolute path
		newFolder := fmt.Sprintf("%s/%s", currentDir, initDir)

		err = os.Mkdir(newFolder, 0755)
		if err != nil {
			fmt.Println("cannot make directory ", err)
			os.Exit(1)
		}

		fmt.Println("Successfully initialized...")
	}

	//for taking snapshots associated wiith the given key
	argMap["snapshot"] = func() {
		if snapshotKey == "" {
			fmt.Println("You should provide a key for identifying the snapshot")
			os.Exit(1)
		}

		currDir, err := os.Getwd()
		if err != nil {
			fmt.Println("current working directory cannot be found...", err)
			os.Exit(1)
		}

		dir, err := os.Open(currDir)
		if err != nil {
			fmt.Println("cannot open the current directory...", err)
			os.Exit(1)
		}
		defer dir.Close()

		entries, err := dir.Readdir(-1)
		if err != nil {
			fmt.Println("cannot read ,the current directory...", err)
			os.Exit(1)
		}
		isninitalized := false
		for _, entry := range entries {
			if entry.IsDir() && entry.Name() == initDir {
				isninitalized = true
			}
		}
		if !isninitalized {
			fmt.Println("manager not initialized...", err)
			os.Exit(1)
		}

		zipfilename := fmt.Sprintf("%s.zip", snapshotKey)
		zipfilename = filepath.Join(initDir, zipfilename)

		zipFile, err := os.Create(zipfilename)
		if err != nil {
			fmt.Println("cannot create zip file...", err)
			os.Exit(1)
		}

		zipWriter := zip.NewWriter(zipFile)
		defer zipWriter.Close()

		filepath.Walk(currDir, func(path string, info fs.FileInfo, err error) error {

			if err != nil {
				fmt.Println("there has been an error", err)
				return err
			}

			file, err := os.Open(path)
			if err != nil {
				fmt.Println("there has been an error in opening the file", err)
				return err
			}
			defer file.Close()

			relativepath, err := filepath.Rel(currDir, path)
			if err != nil {
				fmt.Println("there has been an error in creating the raltive path", err)
				return err
			}

			zipEntry, err := zipWriter.Create(relativepath)
			if err != nil {
				fmt.Println("there has been an error in creaating zipEntry", err)
				return err
			}

			_, err = io.Copy(zipEntry, file)
			if err != nil {
				fmt.Println("there has been an error in copying the file to the zip", err)
				return err
			}

			return nil
		})
	}

}

func main() {

	flag.StringVar(&snapshotKey, "key", "", "unique key for the snapshot")
	flag.Parse()

	set()

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
