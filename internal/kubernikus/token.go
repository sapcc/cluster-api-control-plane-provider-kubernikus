// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package kubernikus

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// GetToken this gets a token from the kubernikus auth service
// it needs to determine the correct url by calling the authUrl and checking for redirects
func GetToken(username, password, connectorId, authUrl string) (string, error) {
	var redirects = 0
	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirects++
			return nil
		},
	}
	req, err := http.NewRequest(http.MethodGet, authUrl, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("failed to build initial request: %w", err)
	}
	q := url.Values{}
	q.Set("connector_id", connectorId)
	req.URL.RawQuery = q.Encode()
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call %s: %w", authUrl, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("calling %s failed with %s, maybe because of an incorrect connector_id", resp.Request.URL, resp.Status)
	}
	if redirects < 1 {
		return "", errors.New("login failed, expected some redirects")
	}
	redirects = 0
	v := url.Values{}
	v.Set("login", username)
	v.Set("password", password)

	resp2, err := client.PostForm(resp.Request.URL.String(), v)
	if err != nil {
		return "", fmt.Errorf("failed to call %s: %w", resp.Request.URL.String(), err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode >= 400 {
		return "", fmt.Errorf("calling %s failed with %s", resp2.Request.URL.String(), resp.Status)
	}
	if redirects < 1 {
		return "", errors.New("login failed, probably because of an incorrect username/password")
	}
	p, err := io.ReadAll(resp2.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read auth response: %w", err)
	}
	var token struct {
		IDToken string `json:"idToken"`
		Type    string `json:"type"`
	}
	if err := json.Unmarshal(p, &token); err != nil {
		return "", errors.New("failed")
	}
	return token.IDToken, nil
}
