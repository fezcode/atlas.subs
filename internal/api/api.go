package api

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Subtitle holds the metadata returned by OpenSubtitles REST API.
type Subtitle struct {
	SubFileName      string `json:"SubFileName"`
	LanguageName     string `json:"LanguageName"`
	SubDownloadLink  string `json:"SubDownloadLink"`
	MovieReleaseName string `json:"MovieReleaseName"`
}

func Search(query string) ([]Subtitle, error) {
	// rest.opensubtitles.org 302-redirects any query containing uppercase
	// letters to a broken URL (host="_"). Lowercase up front to stay on the
	// working code path.
	query = strings.ToLower(query)
	apiURL := fmt.Sprintf("https://rest.opensubtitles.org/search/query-%s", url.PathEscape(query))
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("query=%q url=%q: %w", query, apiURL, err)
	}

	req.Header.Set("User-Agent", "TemporaryUserAgent")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("query=%q url=%q: %w", query, apiURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var subs []Subtitle
	if err := json.NewDecoder(resp.Body).Decode(&subs); err != nil {
		return nil, fmt.Errorf("failed to decode API response: %v", err)
	}

	return subs, nil
}

func DownloadSubtitle(sub Subtitle) (string, error) {
	req, err := http.NewRequest("GET", sub.SubDownloadLink, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "TemporaryUserAgent")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download: status %d", resp.StatusCode)
	}

	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read gzip stream: %v", err)
	}
	defer gzReader.Close()

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	destPath := filepath.Join(cwd, sub.SubFileName)
	outFile, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, gzReader); err != nil {
		return "", fmt.Errorf("failed to save file: %v", err)
	}

	return destPath, nil
}
