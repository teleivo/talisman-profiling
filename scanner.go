package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"runtime/pprof"
	"strings"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	additions := GetAdditions(false)
	_ = additions
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

// BlobsInCommits is a map of blob and list of the commits the blobs is present in.
type BlobsInCommits struct {
	commits map[string][]string
}

// GetAdditions will get all the additions for entire git history
func GetAdditions(ignoreHistory bool) []Addition {
	blobsInCommits := getBlobsInCommit(ignoreHistory)
	var additions []Addition
	for blob := range blobsInCommits.commits {
		objectDetails := strings.Split(blob, "\t")
		objectHash := objectDetails[0]
		data := getData(objectHash)
		filePath := objectDetails[1]
		newAddition := NewScannerAddition(filePath, blobsInCommits.commits[blob], data)
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
	blobs := <-result
	commit := blobs[len(blobs)-1]
	for _, blob := range blobs[:len(blobs)] {
		if blob != "" && blob != commit {
			blobDetailsString := strings.Split(blob, " ")
			blobDetails := strings.Split(blobDetailsString[2], "	")
			blobHash := blobDetails[0] + "\t" + blobDetails[1]
			blobsInCommits.commits[blobHash] = append(blobsInCommits.commits[blobHash], commit)
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
	commits := make(map[string][]string)
	return BlobsInCommits{commits: commits}
}
