package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type funcc func()

// stores all the tags and the tasks associated with the tag
var (
	initDir     = ".manager"
	argMap      = make(map[string]funcc)
	snapshotKey string
)

func main() {

	// flag.StringVar(&snapshotKey, "key", "", "unique key for the snapshot")
	// flag.Parse()

	arguments := os.Args[1:]
	fmt.Println(arguments)
	for i, arg := range arguments {
		if arg == "-key" && i < len(arguments)-1 {
			snapshotKey = arguments[i+1]
			break
		}
	}
	flag.CommandLine.Parse(arguments)

	fmt.Println("here is the key", snapshotKey)

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

		excludeDir := filepath.Join(currDir, initDir)

		filepath.Walk(currDir, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				fmt.Println("there has been an error", err)
				return err
			}

			if path == excludeDir || strings.HasPrefix(path, excludeDir+string(filepath.Separator)) {
				return nil
			}

			relativePath, err := filepath.Rel(currDir, path)
			if err != nil {
				fmt.Println("there has been an error in creating the relative path", err)
				return err
			}

			// doing this to include the directories as well into the zipped archive
			var zipEntry io.Writer
			if info.IsDir() {
				//
				zipEntry, err = zipWriter.CreateHeader(&zip.FileHeader{
					Name:   relativePath + "/",
					Method: zip.Deflate,
				})
				if err != nil {
					fmt.Println("there has been an error in creating directory header", err)
					return err
				}
			} else {

				zipEntry, err = zipWriter.Create(relativePath)
				if err != nil {
					fmt.Println("there has been an error in creating zipEntry", err)
					return err
				}

				file, err := os.Open(path)
				if err != nil {
					fmt.Println("there has been an error in opening the file", err)
					return err
				}
				defer file.Close()

				_, err = io.Copy(zipEntry, file)
				if err != nil {
					fmt.Println("there has been an error in copying the file to the zip", err)
					return err
				}
			}

			return nil
		})

	}

}
