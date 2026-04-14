package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
)

type Platform struct {
	url   string
	name  string
	token string
}

type Mirror struct {
	platforms []Platform
}

func (m *Mirror) add(name, token, url string) {
	m.platforms = append(m.platforms, Platform{
		url:   url,
		name:  name,
		token: token,
	})
}

func getTokens() (map[string]string, error) {
	tokens := make(map[string]string)

	f, err := os.Open(".env")
	if err != nil {
		return nil, fmt.Errorf("env file not found: %w", err)
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

type Payload struct {
	Name       string `json:"name"`
	Private    bool   `json:"private"`
	Path       string `json:"path"`
	Visibility string `json:"visibility"`
}

type RepoData struct {
	Header  map[string]string
	Payload Payload
}

func (p *Platform) createRepo(data RepoData) error {
	fmt.Printf("[*] Creating %s repo on %s: %s\n", data.Payload.Visibility, p.name, data.Payload.Name)

	payload, err := json.Marshal(data.Payload)
	if err != nil {
		return fmt.Errorf("unable to create payload: %w", err)
	}

	req, err := http.NewRequest("POST", p.url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("unable to create http request object: %w", err)
	}

	for k, v := range data.Header {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// body, _ := io.ReadAll(resp.Body)
	// fmt.Printf("[%s] status=%d body=%s\n", p.name, resp.StatusCode, string(body))

	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Printf("[->] %s repo created\n", p.name)
	case http.StatusBadRequest, http.StatusUnprocessableEntity, http.StatusConflict:
		fmt.Printf("[!] repo already exists on %s\n", p.name)
	default:
		fmt.Printf("[!] %s repo creation status code: %d\n", p.name, resp.StatusCode)
	}

	return nil
}

const LOCAL_URL = "https://git.davesaah-pc/api/v1"
const EXTERNAL_USER = "davesaah"
const VERIFY_CERT = "/etc/ssl/homelab/git.davesaah-pc.pem"

func main() {
	var err error

	repoName := "testing-mirror-sync-v2"
	// localOwner := ""
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

	data := RepoData{
		Payload: Payload{
			Name:       repoName,
			Private:    visibility == "private",
			Visibility: visibility,
			Path:       repoName,
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
			if err := p.createRepo(d); err != nil {
				fmt.Println(err)
			}
		})
	}

	wg.Wait()
}

type errOptions struct {
	err  error
	exit bool
}

func newErrOptions() *errOptions {
	return &errOptions{
		exit: false,
	}
}

type ErrOption func(*errOptions)

func WithErr(v error) ErrOption {
	return func(o *errOptions) {
		o.err = v
	}
}

func WithExit(v bool) ErrOption {
	return func(o *errOptions) {
		o.exit = v
	}
}

func check(opts ...ErrOption) {
	o := newErrOptions()
	for _, opt := range opts {
		opt(o)
	}

	if o.err != nil {
		fmt.Println(o.err)
		if o.exit {
			os.Exit(1)
		}
	}
}
