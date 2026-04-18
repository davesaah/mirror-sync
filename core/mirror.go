package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/viper"
)

// Platform defines the credentials + config for interacting with a cloud
// repository provider
type Platform struct {
	url   string
	name  string
	token string
	host  string
}

// Mirror is a list of platforms: github, gitlab and codeberg
type Mirror struct {
	platforms []Platform
}

func (m *Mirror) add(name, token, _url string) {
	urlObj, _ := url.Parse(_url)

	hostParts := strings.Split(urlObj.Hostname(), ".")
	tld := hostParts[len(hostParts)-1]

	m.platforms = append(m.platforms, Platform{
		url:   _url,
		name:  name,
		token: token,
		host:  fmt.Sprintf("%s.%s", name, tld),
	})
}

// createRepo creates a repository on the cloud provider using the RepoData
func (p *Platform) createRepo(data RepoData) error {
	fmt.Printf("[*] Creating %s repo on %s: %s\n", data.Payload.Visibility, p.name, data.Payload.Name)

	payload, _ := json.Marshal(data.Payload)
	client := NewHTTPClient()
	resp, err := client.DoRequest("POST", p.url, payload, data.Header)
	if err != nil {
		return fmt.Errorf("unable to create repo: %w", err)
	}
	defer resp.Body.Close()

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

// sync sets up the created cloud platform repo as a mirror on the local gitea server
func (p *Platform) sync(localOwner, repoName, localToken string) error {
	fmt.Printf("[*] Adding %s mirror...\n", p.name)

	user := viper.GetString("external-user")
	if p.name == "gitlab" {
		user = "oauth2"
	}

	endpoint := fmt.Sprintf("%s/api/v1/repos/%s/%s/push_mirrors", viper.GetString("local-url"), localOwner, repoName)
	remoteUrl := fmt.Sprintf("https://%s:%s@%s/%s/%s.git", user, p.token, p.host, viper.GetString("external-user"), repoName)

	_payload := struct {
		RemoteAddress string `json:"remote_address"`
		SyncOnCommit  bool   `json:"sync_on_commit"`
		Interval      string `json:"interval"`
	}{
		RemoteAddress: remoteUrl,
		SyncOnCommit:  true,
		Interval:      "24h",
	}
	payload, err := json.Marshal(_payload)
	if err != nil {
		return fmt.Errorf("unable to parse payload for syncing mirror: %w", err)
	}

	headers := map[string]string{
		"Authorization": fmt.Sprintf("token %s", localToken),
		"Content-Type":  "application/json",
	}

	client := NewHTTPClient()
	resp, err := client.DoRequest("POST", endpoint, payload, headers)
	if err != nil {
		return fmt.Errorf("unable to add repo as mirror: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[!] failed to add %s mirror", p.name)
	}

	fmt.Printf("[->] %s mirror added successfully\n", p.name)
	return nil
}
