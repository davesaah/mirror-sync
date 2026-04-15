package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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
