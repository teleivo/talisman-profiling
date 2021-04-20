package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

func main() {
	start := time.Now()
	additions := GetAdditions(false)
	_ = additions
	elapsed := time.Since(start)
	fmt.Printf("took %v to get all additions", elapsed)
}

type Addition struct {
	Path    string
	Name    string
	Commits []string
	Data    []byte
}

//NewScannerAddition returns an new Addition for a file with supplied contents and all of the commits the file is in
func NewScannerAddition(filePath string, commits []string, content []byte) Addition {
	return Addition{
		Path:    filePath,
		Name:    filePath,
		Commits: commits,
		Data:    content,
	}
}

type blobDetails struct {
	hash, filePath string
}

// BlobsInCommits is a map of blob and list of the commits the blobs is present in.
type BlobsInCommits struct {
	commits map[blobDetails][]string
}

// GetAdditions will get all the additions for entire git history
func GetAdditions(ignoreHistory bool) []Addition {
	blobsInCommits := getBlobsInCommit(ignoreHistory)
	var additions []Addition
	for blob := range blobsInCommits.commits {
		newAddition := NewScannerAddition(blob.filePath, blobsInCommits.commits[blob], getData(blob.hash))
		additions = append(additions, newAddition)
	}
	return additions
}

func getBlobsInCommit(ignoreHistory bool) BlobsInCommits {
	commits := getAllCommits(ignoreHistory)
	blobsInCommits := newBlobsInCommit()
	result := make(chan []string, len(commits))
	for _, commit := range commits {
		go putBlobsInChannel(commit, result)
	}
	for i := 1; i < len(commits); i++ {
		getBlobsFromChannel(blobsInCommits, result)
	}
	return blobsInCommits
}

func putBlobsInChannel(commit string, result chan []string) {
	if commit != "" {
		blobDetailsBytes, _ := exec.Command("git", "ls-tree", "-r", commit).CombinedOutput()
		blobDetailsList := strings.Split(string(blobDetailsBytes), "\n")
		blobDetailsList = append(blobDetailsList, commit)
		result <- blobDetailsList
	}
}

func getBlobsFromChannel(blobsInCommits BlobsInCommits, result chan []string) {
	blobEntries := <-result
	commit := blobEntries[len(blobEntries)-1]
	for _, blobEntry := range blobEntries[:len(blobEntries)-1] {
		if blobEntry != "" {
			blobHashAndName := strings.Split(strings.Split(blobEntry, " ")[2], "\t")
			blob := blobDetails{hash: blobHashAndName[0], filePath: blobHashAndName[1]}
			blobsInCommits.commits[blob] = append(blobsInCommits.commits[blob], commit)
		}
	}
}

func getAllCommits(ignoreHistory bool) []string {
	commitRange := "--all"
	if ignoreHistory {
		commitRange = "--max-count=1"
	}
	out, err := exec.Command("git", "log", commitRange, "--pretty=%H").CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	return strings.Split(string(out), "\n")
}

func getData(objectHash string) []byte {
	out, _ := exec.Command("git", "cat-file", "-p", objectHash).CombinedOutput()
	return out
}

func newBlobsInCommit() BlobsInCommits {
	commits := make(map[blobDetails][]string)
	return BlobsInCommits{commits: commits}
}
