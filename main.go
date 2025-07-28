package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type GitRepo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func ContainDotGitFolder(files []fs.FileInfo) bool {
	for _, f := range files {
		if f.IsDir() && f.Name() == ".git" {
			return true
		}
	}
	return false
}

func IAmRepoAuthor(repoPath string) bool {
	content, err := os.ReadFile(repoPath)
	if err != nil {
		log.Println("Error: ", err)
		return false
	}
	return strings.Contains(strings.ToLower(string(content)), "pablothedeveloper")

}

func Command(bin string, cmdOps ...string) {
	cmd := exec.Command(bin, cmdOps...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Println(err)
	}
}

func ReadFiles(basePath string) []fs.FileInfo {
	files, err := ioutil.ReadDir(basePath)
	if err != nil {
		log.Fatal(err)
	}
	return files
}

var repos []GitRepo = make([]GitRepo, 0)
var fetchedRepos []GitRepo = make([]GitRepo, 0)

func ExtractGitRepos(basepath string) {
	files := ReadFiles(basepath)
	for _, f := range files {
		path := filepath.Join(basepath, f.Name())
		if f.IsDir() && f.Name() != ".git" && f.Name() != ".cache" {
			if ContainDotGitFolder(ReadFiles(path)) &&
				IAmRepoAuthor(filepath.Join(path, ".git", "config")) {
				repos = append(repos, GitRepo{Name: f.Name(), Path: path})
			} else {
				ExtractGitRepos(path)
			}
		}
	}
}

func GenerateAliasFile(repos []GitRepo) string {
	content := ""
	for _, repo := range repos {
		content = content + fmt.Sprintf("alias %s='cd %s && ls && cat README.md && git pull'\n", repo.Name, repo.Path)
		content = content + fmt.Sprintf("export %s=\"%s\"\n", repo.Name, repo.Path)
	}
	return content
}

func main() {
	// Read datafile
	if _, err := os.Stat("home/dev/.generated_repo_list.json"); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Println("No datafile right now. Will generate one.")
		} else {
			log.Fatal(err)
		}
		// file exists
	} else {
		content, err := os.ReadFile("home/dev/.generated_repo_list.json")
		if err != nil {
			log.Println("Error: ", err)
		}
		if err := json.Unmarshal(content, &fetchedRepos); err != nil {
			log.Fatal(err)
		}
	}
	// extract existing repos
	ExtractGitRepos("/home/dev")

	// find differences
	existing := map[string]GitRepo{}
	for _, repo := range repos {
		existing[repo.Name] = repo
	}
	fetched := map[string]GitRepo{}
	for _, repo := range repos {
		fetched[repo.Name] = repo
	}
	// Clones any repos I need to fetch
	// and merges differences.
	for fetched_key, fetched_val := range fetched {
		if existing_val, ok := existing[fetched_key]; !ok {
			Command("mkdir", "-p", existing_val.Path)
			Command("gh", "repo", "clone", fetched_val.Name, fetched_val.Path)
		} else {
			if existing_val.Path != fetched_val.Path {
				panic("shouldn't happen, but if it does, fix it manually.")
				// TODO: prompt user to delete one and use the other.
			}
		}
	}

	// Pull repos, and pushes commits.
	for existing_key, existing_val := range existing {
		Command("gh", "repo", "sync", fmt.Sprintf("PabloTheDeveloper/%s", existing_val.Name))
		if _, ok := fetched[existing_key]; !ok {
			fetched[existing_key] = existing_val
		}
	}
	synced_repos := []GitRepo{}
	for _, v := range fetched {
		synced_repos = append(synced_repos, v)
	}

	// Ensures consistent order
	sort.Slice(synced_repos, func(i, j int) bool {
		return synced_repos[i].Name < synced_repos[j].Name
	})
	// Create datafile
	if jsonData, err := json.Marshal(synced_repos); err != nil {
		log.Fatal(err)
	} else {
		err = os.WriteFile("/home/dev/.generated_repo_list.json", jsonData, 0644)
		if err != nil {
			log.Fatal(err)
		}
	}
	// TODO: Create Repos which don't exist in current location.

	// TODO: Pick between conflicting synced_repos locations.

	content := GenerateAliasFile(synced_repos)
	err := os.WriteFile("/home/dev/.generated_repo_aliases", []byte(content), 0644)
	if err != nil {
		log.Fatal(err)
	}

}
