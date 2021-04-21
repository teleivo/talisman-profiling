package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	_ "net/http/pprof"

	"github.com/google/go-cmp/cmp"
)

var test = flag.Bool("test", false, "compare result of sequential and concurrent approach")
var sequential = flag.Bool("sequential", false, "get blobs in commits sequentially")

// var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	// To test that the sequential version gathers the same blobs in commits
	if *test {
		blobsInCommitsSequential := getBlobsInCommitSequential(false)
		blobsInCommits := getBlobsInCommit(false)
		if diff := cmp.Diff(blobsInCommits, blobsInCommitsSequential); diff != "" {
			fmt.Println(fmt.Errorf("blobs mismatch (-want +got):\n%s", diff))
			os.Exit(1)
		}
		os.Exit(0)
	}

	go http.ListenAndServe(":1234", nil)
	// if *cpuprofile != "" {
	// 	f, err := os.Create(*cpuprofile)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	pprof.StartCPUProfile(f)
	// 	defer pprof.StopCPUProfile()
	// }
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
	// So that we can run the original version using goroutines and channels
	// and the sequential one for comparison
	var blobsInCommits BlobsInCommits
	if *sequential {
		blobsInCommits = getBlobsInCommitSequential(ignoreHistory)
	} else {
		blobsInCommits = getBlobsInCommit(ignoreHistory)
	}
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

func getBlobsInCommitSequential(ignoreHistory bool) BlobsInCommits {
	commits := getAllCommits(ignoreHistory)
	blobsInCommits := newBlobsInCommit()
	blobsPerCommit := make([][]string, len(commits))
	for _, commit := range commits {
		if commit != "" {
			blobDetailsBytes, _ := exec.Command("git", "ls-tree", "-r", commit).CombinedOutput()
			blobDetailsList := strings.Split(string(blobDetailsBytes), "\n")
			blobDetailsList = append(blobDetailsList, commit)
			blobsPerCommit = append(blobsPerCommit, blobDetailsList)
		}
	}
	for _, blobs := range blobsPerCommit {
		if len(blobs) == 0 {
			continue // needed since commits has an empty "" in its slice
			// this guard clause is the equivalent of the i:=1 in
			// for i := 1; i < len(commits); i++ {
			// 	getBlobsFromChannel(blobsInCommits, result)
			// }
		}
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
	return blobsInCommits
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
