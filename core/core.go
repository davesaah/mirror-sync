package core

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

type LocalRepoInfo struct {
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

func fetchLocalRepoInfo(owner, repoName, token string) (*LocalRepoInfo, error) {
	fmt.Println("[*] Fetching local repo info from Gitea...")
	endpoint := fmt.Sprintf("%s/repos/%s/%s", LOCAL_URL, owner, repoName)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create http request object: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%s repo does not exist on local server", repoName)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch local info body for %s repo: %w", repoName, err)
	}

	fmt.Printf("body: %s\n", string(body))

	var info LocalRepoInfo
	json.Unmarshal(body, &info)

	info.Description = fmt.Sprintf("[Mirror] %s", info.Description)

	return &info, nil
}

func Run(repoName, localOwner, visibility string) error {
	var wg sync.WaitGroup

	tokens, err := getTokens()
	if err != nil {
		return err
	}

	mirrors := Mirror{}
	mirrors.add("github", tokens["GITHUB_TOKEN"], "https://api.github.com/user/repos")
	mirrors.add("gitlab", tokens["GITLAB_TOKEN"], "https://gitlab.com/api/v4/projects")
	mirrors.add("codeberg", tokens["CODEBERG_TOKEN"], "https://codeberg.org/api/v1/user/repos")

	localInfo, err := fetchLocalRepoInfo(localOwner, repoName, tokens["LOCALHOST_TOKEN"])
	if err != nil {
		return err
	}

	data := RepoData{
		Payload: Payload{
			Name:        repoName,
			Private:     visibility == "private",
			Visibility:  visibility,
			Path:        repoName,
			Description: localInfo.Description,
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
			if err != nil {
				log.Println(err)
			}
			err = p.sync(localOwner, repoName, tokens["LOCALHOST_TOKEN"])
			if err != nil {
				log.Println(err)
			}
		})
	}

	wg.Wait()
	return nil
}
