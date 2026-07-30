package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
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

type speakerReference struct {
	Name   string `json:"name"`
	Sample string `json:"sample"`
}

func speakerReferenceDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "transcriber", "speaker_references"), nil
}

func validateSpeakerName(name string) error {
	if name == "" || strings.TrimSpace(name) != name {
		return fmt.Errorf("invalid speaker name")
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == ' ' || r == '-' || r == '_') {
			return fmt.Errorf("invalid speaker name")
		}
	}
	return nil
}

func speakerReferencePaths(name, sample string) (string, string, error) {
	if err := validateSpeakerName(name); err != nil {
		return "", "", err
	}
	if sample != "" && (filepath.Base(sample) != sample || sample == "." || sample == "..") {
		return "", "", fmt.Errorf("invalid speaker sample")
	}
	dir, err := speakerReferenceDir()
	if err != nil {
		return "", "", err
	}
	return filepath.Join(dir, name+".json"), filepath.Join(dir, sample), nil
}

func ensureSpeakerReferenceDir() (string, error) {
	dir, err := speakerReferenceDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, os.Chmod(dir, 0700)
}

func saveSpeakerReference(name, source string) error {
	if err := validateSpeakerName(name); err != nil {
		return err
	}
	ffprobe, err := lookPath("ffprobe")
	if err != nil {
		return err
	}
	durationOutput, err := exec.Command(ffprobe, "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", source).CombinedOutput()
	if err != nil {
		if output := strings.TrimSpace(string(durationOutput)); output != "" {
			return fmt.Errorf("probing speaker reference duration: %s", output)
		}
		return fmt.Errorf("probing speaker reference duration: %w", err)
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(string(durationOutput)), 64)
	if err != nil {
		return fmt.Errorf("parsing speaker reference duration: %w", err)
	}
	if !(duration >= 1.2 && duration <= 10) {
		return fmt.Errorf("speaker reference duration %.1fs outside required range 1.2s-10s", duration)
	}
	if _, err := ensureSpeakerReferenceDir(); err != nil {
		return err
	}
	sample := name + ".sample" + filepath.Ext(source)
	metadataPath, samplePath, err := speakerReferencePaths(name, sample)
	if err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(samplePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.Chmod(samplePath, 0600); err != nil {
		return err
	}
	metadata, err := json.Marshal(speakerReference{Name: name, Sample: sample})
	if err != nil {
		return err
	}
	if err := os.WriteFile(metadataPath, metadata, 0600); err != nil {
		return err
	}
	return os.Chmod(metadataPath, 0600)
}

func loadSpeakerReference(name string) (speakerReference, error) {
	metadataPath, _, err := speakerReferencePaths(name, "")
	if err != nil {
		return speakerReference{}, err
	}
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return speakerReference{}, err
	}
	var reference speakerReference
	if err := json.Unmarshal(data, &reference); err != nil {
		return speakerReference{}, err
	}
	if reference.Name != name || filepath.Base(reference.Sample) != reference.Sample || !strings.HasPrefix(reference.Sample, name+".sample") || reference.Sample == name+".json" {
		return speakerReference{}, fmt.Errorf("invalid speaker reference")
	}
	return reference, nil
}

func listSpeakerReferences() ([]string, error) {
	dir, err := speakerReferenceDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		if _, err := loadSpeakerReference(name); err == nil {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

func removeSpeakerReference(name string) error {
	reference, err := loadSpeakerReference(name)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	metadataPath, samplePath, err := speakerReferencePaths(name, reference.Sample)
	if err != nil {
		return err
	}
	if err := os.Remove(samplePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
