package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".transcriber_key")
}

func loadAPIKey() (string, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func saveAPIKey(key string) error {
	return os.WriteFile(configPath(), []byte(key), 0600)
}

func getAPIKey() (string, error) {
	if key, err := loadAPIKey(); err == nil && key != "" {
		return key, nil
	}
	fmt.Fprint(os.Stderr, "OpenAI API key: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	key := strings.TrimSpace(scanner.Text())
	if key == "" {
		return "", fmt.Errorf("API key required")
	}
	if err := saveAPIKey(key); err != nil {
		return "", fmt.Errorf("saving key: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Key saved to ~/.transcriber_key")
	return key, nil
}
