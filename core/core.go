package core

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/spf13/viper"
)

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

func fetchLocalRepoInfo(owner, repoName, token string) (*LocalRepoInfo, error) {
	fmt.Println("[*] Fetching local repo info from Gitea...")
	endpoint := fmt.Sprintf("%s/api/v1/repos/%s/%s", viper.GetString("local-url"), owner, repoName)

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

	var info LocalRepoInfo
	json.Unmarshal(body, &info)

	info.Description = fmt.Sprintf("[Mirror] %s", info.Description)

	return &info, nil
}

func Run(repoName, localOwner, visibility string) error {
	var wg sync.WaitGroup

	mirrors := Mirror{}
	mirrors.add("github", viper.GetString("github-token"), "https://api.github.com/user/repos")
	mirrors.add("gitlab", viper.GetString("gitlab-token"), "https://gitlab.com/api/v4/projects")
	mirrors.add("codeberg", viper.GetString("codeberg-token"), "https://codeberg.org/api/v1/user/repos")

	localInfo, err := fetchLocalRepoInfo(localOwner, repoName, viper.GetString("localhost-token"))
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
			err = p.sync(localOwner, repoName, viper.GetString("localhost-token"))
			if err != nil {
				log.Println(err)
			}
		})
	}

	wg.Wait()
	return nil
}
