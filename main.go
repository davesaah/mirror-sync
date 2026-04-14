package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

type Platform struct {
	url   string
	name  string
	token string
	host  string
}

type Mirror struct {
	platforms []Platform
}

func (m *Mirror) add(name, token, _url string) error {
	urlObj, err := url.Parse(_url)
	if err != nil {
		return fmt.Errorf("invalid mirror url: %w", err)
	}

	hostParts := strings.Split(urlObj.Hostname(), ".")
	tld := hostParts[len(hostParts)-1]

	m.platforms = append(m.platforms, Platform{
		url:   _url,
		name:  name,
		token: token,
		host:  fmt.Sprintf("%s.%s", name, tld),
	})

	return nil
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

func (p *Platform) sync(localOwner, repoName, localToken string) error {
	fmt.Printf("[*] Adding %s mirror...\n", p.name)

	user := EXTERNAL_USER
	if p.name == "gitlab" {
		user = "oauth2"
	}

	endpoint := fmt.Sprintf("%s/repos/%s/%s/push_mirrors", LOCAL_URL, localOwner, repoName)
	remoteUrl := fmt.Sprintf("https://%s:%s@%s/%s/%s.git", user, p.token, p.host, EXTERNAL_USER, repoName)

	_payload := struct {
		RemoteAddress string `json:"remote_address"`
		SyncOnCommit  bool   `json:"sync_on_commit"`
		Interval      string `json:"interval"`
	}{
		RemoteAddress: remoteUrl,
		SyncOnCommit:  false,
		Interval:      "24h",
	}
	payload, err := json.Marshal(_payload)
	if err != nil {
		return fmt.Errorf("unable to parse payload for syncing mirror: %w", err)
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("unable to create http request object: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", localToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// body, _ := io.ReadAll(resp.Body)
	// fmt.Printf("[%s] status=%d body=%s\n", p.name, resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[!] failed to add %s mirror\n", p.name)
	}

	fmt.Printf("[->] %s mirror added successfully\n", p.name)
	return nil
}

const LOCAL_URL = "https://git.davesaah-pc/api/v1"
const EXTERNAL_USER = "davesaah"
const VERIFY_CERT = "/etc/ssl/homelab/git.davesaah-pc.pem"

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
			err = p.createRepo(d)
			check(WithErr(err))

			err = p.sync(localOwner, repoName, tokens["LOCALHOST_TOKEN"])
			check(WithErr(err))
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
