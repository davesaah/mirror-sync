package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
)

const LOCAL_URL = "https://git.davesaah-pc/api/v1"
const EXTERNAL_USER = "davesaah"

type Payload struct {
	Name        string `json:"name"`
	Private     bool   `json:"private"`
	Path        string `json:"path"`
	Visibility  string `json:"visibility"`
	Description string `json:"description"`
}

type RepoData struct {
	Header  map[string]string
	Payload Payload
}

type DescriptionResponse struct {
	Description string `json:"description"`
}

func getTokens() (map[string]string, error) {
	tokens := make(map[string]string)

	f, err := os.Open(".env")
	if err != nil {
		return nil, fmt.Errorf("unable to get tokens. env file not found: %w", err)
	}

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "=", 2)
		if len(parts) == 2 {
			tokens[parts[0]] = parts[1]
		}
	}

	return tokens, nil
}

func fetchDescription(owner, repoName, token string) (string, error) {
	fmt.Println("[*] Fetching description from Gitea...")
	endpoint := fmt.Sprintf("%s/repos/%s/%s", LOCAL_URL, owner, repoName)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("unable to create http request object: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("unable to fetch description for %s repo: %w", repoName, err)
	}

	var bodyObj DescriptionResponse
	if err = json.Unmarshal(body, &bodyObj); err != nil {
		return "", fmt.Errorf("unable to fetch description for %s repo: %w", repoName, err)
	}

	return fmt.Sprintf("[Mirror] %s", bodyObj.Description), nil
}

func main() {
	var err error

	repoName := "testing-mirror-sync-v2"
	localOwner := "davesaah"
	visibility := "private"

	var wg sync.WaitGroup

	tokens, err := getTokens()
	check(WithErr(err), WithExit(true))

	mirrors := Mirror{}
	err = mirrors.add("github", tokens["GITHUB_TOKEN"], "https://api.github.com/user/repos")
	check(WithErr(err))
	err = mirrors.add("gitlab", tokens["GITLAB_TOKEN"], "https://gitlab.com/api/v4/projects")
	check(WithErr(err))
	err = mirrors.add("codeberg", tokens["CODEBERG_TOKEN"], "https://codeberg.org/api/v1/user/repos")
	check(WithErr(err))
	description, err := fetchDescription(localOwner, repoName, tokens["LOCALHOST_TOKEN"])
	check(WithErr(err), WithExit(true))

	data := RepoData{
		Payload: Payload{
			Name:        repoName,
			Private:     visibility == "private",
			Visibility:  visibility,
			Path:        repoName,
			Description: description,
		},
	}

	for _, platform := range mirrors.platforms {
		p := platform
		d := RepoData{
			Payload: data.Payload,
			Header:  make(map[string]string),
		}

		d.Header["Content-Type"] = "application/json"

		switch p.name {
		case "github":
			d.Header["Authorization"] = fmt.Sprintf("token %s", platform.token)
			d.Header["Accept"] = "application/vnd.github+json"
		case "gitlab":
			d.Header["PRIVATE-TOKEN"] = platform.token
		case "codeberg":
			d.Header["Authorization"] = fmt.Sprintf("token %s", platform.token)
		}

		wg.Go(func() {
			err = p.createRepo(d)
			check(WithErr(err))

			err = p.sync(localOwner, repoName, tokens["LOCALHOST_TOKEN"])
			check(WithErr(err))
		})
	}

	wg.Wait()
}
