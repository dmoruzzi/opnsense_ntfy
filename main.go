package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const confFile = "config.yml"

type RSS struct {
	Items []struct {
		ID      string `xml:"id"`
		Subject string `xml:"subject"`
		Body    string `xml:"body"`
		Link    string `xml:"link"`
	} `xml:"recent-post"`
}

type ServerConfig struct {
	Token      string `yaml:"token"`
	NtfyServer string `yaml:"ntfy_server"`
}

type NTFYConfig struct {
	Servers         map[string]ServerConfig `yaml:"servers"`
	LastSeenID      string                  `yaml:"last_seen_id"`
	FeedURL         string                  `yaml:"feed_url"`
	RefreshInterval string                  `yaml:"refresh_interval"`
}

func fetchFeed(url string) (*RSS, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp.Body); err != nil {
		return nil, err
	}

	var feed RSS
	if err := xml.NewDecoder(&buf).Decode(&feed); err != nil {
		return nil, err
	}
	return &feed, nil
}

func notify(msg string, ntfyEndpoint string, authToken string) error {
	client := &http.Client{}
	req, err := http.NewRequest("POST", ntfyEndpoint, bytes.NewBufferString(msg))
	if err != nil {
		return err
	}
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return fmt.Errorf("notification error: %v", err)
	}
	return nil
}

func readConfig() (map[string]string, string, string, time.Duration, error) {
	var config NTFYConfig
	data, err := os.ReadFile(confFile)
	if err != nil {
		return nil, "", "", 0, err
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, "", "", 0, err
	}

	serverMap := make(map[string]string)
	for _, serverConfig := range config.Servers {
		serverMap[serverConfig.NtfyServer] = serverConfig.Token
	}
	lastSeenID := config.LastSeenID
	feedURL := config.FeedURL
	refreshInterval, err := time.ParseDuration(config.RefreshInterval)
	if err != nil {
		return nil, "", "", 0, err
	}
	return serverMap, lastSeenID, feedURL, refreshInterval, nil
}

func checkForUpdates(lastID string, serverMap map[string]string, feedURL string) (string, error) {
	feed, err := fetchFeed(feedURL)
	if feed.Items[0].ID == lastID {
		log.Printf("No updates. LastID: %s, newID: %s", lastID, feed.Items[0].ID)
		return lastID, err
	} else if len(feed.Items) == 0 {
		return "", fmt.Errorf("no items in feed")
	} else if err != nil {
		return "", err
	}
	latest := feed.Items[0]
	msg := fmt.Sprintf("New update: %s\n%s\n%s", latest.Subject, latest.Link, latest.Body)

	for server, authToken := range serverMap {
		if err := notify(msg, server, authToken); err != nil {
			log.Printf("Error sending notification to %s: %v", server, err)
		}
	}

	writeLastSeenID(latest.ID)
	return latest.ID, nil
}

func writeLastSeenID(id string) {
	config, err := os.ReadFile(confFile)
	if err != nil {
		log.Fatal("Error reading configuration file:", err)
	}

	var parsedConfig NTFYConfig
	if err := yaml.Unmarshal(config, &parsedConfig); err != nil {
		log.Fatal("Error parsing configuration:", err)
	}

	parsedConfig.LastSeenID = id

	newConfig, err := yaml.Marshal(&parsedConfig)
	if err != nil {
		log.Fatal("Error marshaling configuration:", err)
	}

	if err := os.WriteFile(confFile, newConfig, 0644); err != nil {
		log.Fatal("Error writing configuration:", err)
	}
}

func main() {
	serverMap, lastID, feedURL, refreshInterval, err := readConfig()
	if err != nil {
		log.Fatal("Error reading configuration:", err)
	}

	for {
		if newID, err := checkForUpdates(lastID, serverMap, feedURL); err == nil {
			lastID = newID
		} else {
			log.Println("Error:", err)
		}
		time.Sleep(refreshInterval)
	}
}
