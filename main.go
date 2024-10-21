package main

import (
	"encoding/xml"
	"fmt"
	"github.com/go-git/go-git/v5"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
)

const rootDir = "/Users/peter/git/gearset-pipelines-testing-ground"
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

type SfCustomField struct {
	FullName   string `xml:"fullName"`
	ExternalId bool   `xml:"externalId"`
	Label      string `xml:"label"`
	Type       string `xml:"type"`
	OverlayId  string `xml:"x-gs-overlay,attr,omitempty"`
	TestFlag   bool   `xml:"x-gs-devobject,attr,omitempty"`
	Comment    string `xml:",comment"`
	Xmlns      string `xml:"xmlns,attr"`
}

func randomString(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		sb.WriteByte(charset[rand.Intn(len(charset))])
	}
	return sb.String()

}

func handleFieldFile(path string, mode fs.FileMode, workTree *git.Worktree) error {
	// Attempt to parse the XML file
	absPath := fmt.Sprintf("%s/%s", rootDir, path)

	data, err := os.ReadFile(absPath)
	if err != nil {
		log.Printf("Encountered readErr: %+v", err)
		return err
	}

	out := SfCustomField{}
	err = xml.Unmarshal(data, &out)
	if err != nil {
		return nil
	}

	if !out.TestFlag {
		return nil
	}

	if out.Type != "AutoNumber" {
		log.Println("Skipping as only AutoNumbers are supported")
		return nil
	}

	newName := randomString(8)

	log.Printf("Changing label from %s to %s", out.Label, newName)

	out.Label = newName

	newObject, err := xml.MarshalIndent(out, "", "\t")
	if err != nil {
		return err
	}

	output := fmt.Sprintf("%s%s", xml.Header, newObject)

	os.WriteFile(absPath, []byte(output), mode)
	_, err = workTree.Add(path)

	return err
}

func main() {
	rand.Seed(time.Now().UnixNano())
	fileSystem := os.DirFS(rootDir)

	repo, err := git.PlainOpen(rootDir)
	if err != nil {
		log.Fatalf("Can't open repository: %+v", err)
	}
	workTree, err := repo.Worktree()
	if err != nil {
		log.Fatalf("Can't open workTree: %+v", err)
	}

	err = fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		// log.Printf("Searching Path: %s", path)
		if d.IsDir() {
			return nil
		}

		if !strings.Contains(path, "fields") {
			return nil
		}

		if d.Name()[0] == '.' {
			return nil
		}

		// log.Printf("Found object: %s", path)

		// In the future, we might want to push one of these to a channel so we can process lots of files in parallel
		finfo, err := d.Info()
		if err != nil {
			return err
		}
		return handleFieldFile(path, finfo.Mode()&os.ModePerm, workTree)
	})
	if err != nil {
		log.Fatalf("Error: %+v", err)
	}

	status, err := workTree.Status()
	if err != nil {
		log.Fatalf("Error: %+v", err)
	}

	fmt.Println(status)

	commit, err := workTree.Commit("Automated change to prod GS Pipeline", &git.CommitOptions{})
	if err != nil {
		log.Fatalf("Error: %+v", err)
	}

	obj, err := repo.CommitObject(commit)
	if err != nil {
		log.Fatalf("Error: %+v", err)
	}
	fmt.Println(obj)

}
