package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type funcc func()

type ConfigStore struct {
	ProjectID  string     `json:"project_id"`
	Progresses []Progress `json:"progresses"`
	StartIndex int        `json:"start_index"`
}

type Progress struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	Filename    string `json:"file_name"`
}

type SetUserCredentials struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type GetSnapshot struct {
	Snap []byte `json:"snap"`
}

// type RemoteProgress struct {
// 	Email           string        `json:"email"`
// 	ProjectID       string        `json:"project_id"`
// 	ProgressWithZip []ProgressZip `json:"progress_with_zip"`
// }

// type ProgressZip struct {
// 	Progress    int     `json:"progress"`
// 	Key         string  `json:"key"`
// 	Description string  `json:"description"`
// 	Zip         os.File `json:"zip"`
// }

// stores all the tags and the tasks associated with the tag
var (
	initDir             = ".manager"
	argMap              = make(map[string]funcc)
	snapshotKey         string
	snapshotDescription string
	projectID           string
	projectConf         = "configs.json"
	setEmail            string
	setName             string
	userCreds           = ".manager.json"
	isStaged            string
	commitID            string
	ignoreManager       = "manager"
)

func main() {

	// flag.StringVar(&snapshotKey, "key", "", "unique key for the snapshot")
	// flag.Parse()

	arguments := os.Args[1:]
	for i, arg := range arguments {
		if arg == "-key" && i < len(arguments)-1 {
			snapshotKey = arguments[i+1]
		}
		if arg == "init" && i < len(arguments)-1 {
			projectID = arguments[i+1]
			break
		}
		if arg == "-desc" && i < len(arguments)-1 {
			snapshotDescription = arguments[i+1]
			break
		}
		if arg == "-email" && i < len(arguments)-1 {
			setEmail = arguments[i+1]
		}
		if arg == "-name" && i < len(arguments)-1 {
			setName = arguments[i+1]
		}
		if arg == "-stage" && i < len(arguments)-1 {
			isStaged = "true"
		}
		if arg == "-commitID" && i < len(arguments)-1 {
			commitID = arguments[i+1]
		}
	}
	flag.CommandLine.Parse(arguments)

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

		if projectID == "" {
			fmt.Println("please provide the projectID as well...")
			os.Exit(1)
		}

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

		if err = os.Chdir(initDir); err != nil {
			fmt.Println("cannot make directory ", err)
			os.Exit(1)
		}

		configFile, err := os.Create(projectConf)
		if err != nil {
			fmt.Println("cannot make conf file ", err)
			os.Exit(1)
		}
		c := ConfigStore{
			ProjectID: projectID,
		}

		res, err := json.Marshal(c)
		if err != nil {
			fmt.Println("cannot write to file ", err)
			os.Exit(1)
		}
		configFile.Write(res)

		fmt.Println("Successfully initialized...")
	}

	//for taking snapshots associated wiith the given key
	argMap["snapshot"] = func() {
		if snapshotKey == "" {
			fmt.Println("You should provide a key for identifying the snapshot")
			os.Exit(1)
		}

		if snapshotDescription == "" {
			fmt.Println("You should provide a description for the snapshot")
			os.Exit(1)
		}

		zipName := uuid.New().String()

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

		zipfilenamee := fmt.Sprintf("%s.zip", zipName)
		zipfilename := filepath.Join(initDir, zipfilenamee)

		zipFile, err := os.Create(zipfilename)
		if err != nil {
			fmt.Println("cannot create zip file...", err)
			os.Exit(1)
		}

		zipWriter := zip.NewWriter(zipFile)
		defer zipWriter.Close()

		excludeDir := filepath.Join(currDir, initDir)
		excludeGit := filepath.Join(currDir, ".git")
		excludeManager := filepath.Join(currDir, ignoreManager)

		filepath.Walk(currDir, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				fmt.Println("there has been an error", err)
				return err
			}

			if path == excludeDir || strings.HasPrefix(path, excludeDir+string(filepath.Separator)) || path == excludeGit || strings.HasPrefix(path, excludeGit+string(filepath.Separator)) {
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
				_, err = zipWriter.CreateHeader(&zip.FileHeader{
					Name:   relativePath + "/",
					Method: zip.Deflate,
				})
				if err != nil {
					fmt.Println("there has been an error in creating directory header", err)
					return err
				}
			} else {

				if excludeManager == path {
					return nil
				}

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

		c := Progress{
			Key:         snapshotKey,
			Description: snapshotDescription,
			Filename:    zipfilenamee,
		}

		var cp ConfigStore

		if err = os.Chdir(initDir); err != nil {
			fmt.Println("there has been an error in going to the init dir file", err)
			return
		}

		file, err := os.Open(projectConf)
		if err != nil {
			fmt.Println("there has been an error opening the config.json file", err)
			return
		}
		defer file.Close()

		if err := json.NewDecoder(file).Decode(&cp); err != nil {
			fmt.Println("there has been an error decoding the config.json file", err)
			return
		}

		cp.Progresses = append(cp.Progresses, c)

		confFile, err := os.Create(projectConf)
		if err != nil {
			fmt.Println("there has been an error creating the config.json file", err)
			return
		}
		defer confFile.Close()

		if err = json.NewEncoder(confFile).Encode(cp); err != nil {
			fmt.Println("there has been an error writing to the config.json file", err)
			return
		}
	}

	//set email and name of the user
	argMap["set"] = func() {

		if setEmail == "" {
			fmt.Println("the email field cannot be empty")
			return
		}
		if setName == "" {
			fmt.Println("the name field cannot be empty")
			return
		}

		c := SetUserCredentials{
			Email: setEmail,
			Name:  setName,
		}

		usr, err := user.Current()
		if err != nil {
			fmt.Println("failed to get the current user ", err)
			return
		}

		if err = os.Chdir(usr.HomeDir); err != nil {
			fmt.Println("failed to load the current user home dir ", err)
			return
		}

		file, err := os.Create(userCreds)
		if err != nil {
			fmt.Println("failed to create a manager directory in the current user home dir ", err)
			return
		}

		if err = json.NewEncoder(file).Encode(c); err != nil {
			fmt.Println("failed to write to the manager directory in the current user home dir ", err)
			return
		}
	}

	argMap["push"] = func() {

		currDir, err := os.Getwd()
		if err != nil {
			fmt.Println("failed to get the current working directory ", err)
			return
		}

		usr, err := user.Current()
		if err != nil {
			fmt.Println("failed to get the current user ", err)
			return
		}

		if err = os.Chdir(usr.HomeDir); err != nil {
			fmt.Println("failed to switch to the current user directory", err)
			return
		}

		credFile, err := os.Open(userCreds)
		if err != nil {
			fmt.Println("failed to get the creds from the current user directory or the run manager set with your email and name for initializing it ", err)
			return
		}

		var c SetUserCredentials
		if err = json.NewDecoder(credFile).Decode(&c); err != nil {
			fmt.Println("failed to decode the creds from the current user directory ", err)
			return
		}

		if err = os.Chdir(fmt.Sprintf("%s/.manager", currDir)); err != nil {
			fmt.Println("failed to change to the manager dir if not initialized manager please initialize it using manager init ", err)
			return
		}

		conf, err := os.Open(projectConf)
		if err != nil {
			fmt.Println("failed to get the conf file ", err)
			return
		}

		var cc ConfigStore
		if err = json.NewDecoder(conf).Decode(&cc); err != nil {
			fmt.Println("failed to decode the conf file ", err)
			return
		}

		if isStaged == "true" && snapshotKey == "" {
			fmt.Println("the key cannot be empty... ", err)
			return
		}

		var i int
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		writer.WriteField("email", c.Email)
		writer.WriteField("project_id", cc.ProjectID)
		writer.WriteField("isStaged", isStaged)
		writer.WriteField("key", snapshotKey)

		if cc.StartIndex == len(cc.Progresses) {
			fmt.Println("no new commits ", err)
			return
		}

		for i = cc.StartIndex; i < len(cc.Progresses); i++ {

			zipFile, err := os.Open(cc.Progresses[i].Filename)
			if err != nil {
				fmt.Println("failed to open the zip file ", err)
				return
			}
			defer zipFile.Close()

			part, err := writer.CreateFormFile("files", cc.Progresses[i].Filename)
			if err != nil {
				fmt.Println("failed to create form file ", err)
				return
			}

			_, err = io.Copy(part, zipFile)
			if err != nil {
				fmt.Println("failed to copy zip file contents to form file ", err)
				return
			}

			writer.WriteField("keys", cc.Progresses[i].Key)
			writer.WriteField("descriptions", cc.Progresses[i].Description)
			writer.WriteField("progresses", strconv.Itoa(i+1))
		}

		err = writer.Close()
		if err != nil {
			fmt.Println("failed to close multipart writer ", err)
			return
		}

		cc.StartIndex = i

		filee, err := os.Create(projectConf)
		if err != nil {
			fmt.Println("failed to create the zip file ", err)
			return
		}

		if err = json.NewEncoder(filee).Encode(cc); err != nil {
			fmt.Println("failed to encode the zip file ", err)
			return
		}

		resp, err := http.Post("http://localhost:50000/snapshots/push", writer.FormDataContentType(), &buf)

		// req, err := http.NewRequest("POST", "http://localhost:50000/snapshots/push", &buf)
		// if err != nil {
		// 	fmt.Println("failed to create HTTP request:", err)
		// 	return
		// }

		// req.Header.Set("Content-Type", writer.FormDataContentType())
    
		fmt.Println("pushiinggg")
		if err != nil {
			fmt.Println("failed to send HTTP request:", err)
			return
		}
		defer resp.Body.Close()
		fmt.Println("pushed...")
		if resp.StatusCode != http.StatusOK {
			fmt.Println("unexpected response status code:", resp.StatusCode)
			return
		}

		fmt.Println(isStaged)

		fmt.Println("pushing completed successfully...")
	}

	argMap["pull"] = func() {

		if commitID == "" {
			fmt.Println("The commitID cannot be empty...")
			return
		}

		url := fmt.Sprintf("http://localhost:50000/snapshots/pull?commitID=%s", commitID)

		resp, err := http.Get(url)
		if err != nil {
			fmt.Println(err)
			return
		}

		var result GetSnapshot
		if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
			fmt.Println(err)
			return
		}

		zipReader, err := zip.NewReader(bytes.NewReader(result.Snap), int64(len(result.Snap)))
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, file := range zipReader.File {
			filePath := filepath.Join(".", file.Name)

			if file.FileInfo().IsDir() {

				if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
					fmt.Println(err)
					return
				}
			} else {

				outputFile, err := os.Create(filePath)
				if err != nil {
					fmt.Println(err)
					return
				}

				defer outputFile.Close()

				fileReader, err := file.Open()

				if err != nil {
					if err == fs.ErrPermission || err == os.ErrPermission {
						fmt.Println("permission Denied!!!")
						continue
					}
					fmt.Println(err)
					return
				}
				defer fileReader.Close()

				if _, err := io.Copy(outputFile, fileReader); err != nil {
					fmt.Println(err)
					return
				}
			}
		}

		fmt.Println("Successfully pulled the Snapshot from the remote repository")

	}

}
