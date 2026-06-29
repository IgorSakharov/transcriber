package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const (
	maxSize        = 24 * 1024 * 1024 // 24MB, under Whisper's 25MB limit
	whisperTimeout = 5 * time.Minute
	mergeTimeout   = 2 * time.Minute
	maxConcurrent  = 3 // ponytail: cap parallel Whisper calls, raise if account has higher RPM tier
	maxRetries     = 3
)

// validExts is the set Whisper accepts
var validExts = map[string]bool{
	".mp3": true, ".mp4": true, ".mpeg": true, ".mpga": true,
	".m4a": true, ".wav": true, ".webm": true, ".ogg": true, ".flac": true,
}

func transcribeFile(apiKey, path string) (string, error) {
	if err := validateInput(path); err != nil {
		return "", err
	}

	info, _ := os.Stat(path)
	if info.Size() <= maxSize {
		fmt.Fprintln(os.Stderr, "Transcribing...")
		return whisperWithRetry(apiKey, path)
	}

	chunks, cleanup, err := splitFile(path)
	if err != nil {
		return "", err
	}
	defer cleanup()

	fmt.Fprintf(os.Stderr, "Transcribing %d chunks...\n", len(chunks))
	parts := make([]string, len(chunks))
	errs := make([]error, len(chunks))

	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	for i, chunk := range chunks {
		wg.Add(1)
		go func(i int, chunk string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			parts[i], errs[i] = whisperWithRetry(apiKey, chunk)
			fmt.Fprintf(os.Stderr, "  [%d/%d] done\n", i+1, len(chunks))
		}(i, chunk)
	}
	wg.Wait()

	for i, e := range errs {
		if e != nil {
			return "", fmt.Errorf("chunk %d: %w", i+1, e)
		}
	}

	fmt.Fprintln(os.Stderr, "Merging...")
	return mergeChunks(apiKey, parts)
}

func validateInput(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", path)
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory, not a file", path)
	}
	ext := strings.ToLower(filepath.Ext(path))
	if !validExts[ext] {
		return fmt.Errorf("unsupported file type %q — accepted: mp3, mp4, m4a, wav, flac, ogg, webm", ext)
	}
	return nil
}

func whisperWithRetry(apiKey, path string) (string, error) {
	var err error
	for attempt := range maxRetries {
		var text string
		text, err = whisperCall(apiKey, path)
		if err == nil {
			return text, nil
		}
		if !isRetryable(err) {
			return "", err
		}
		wait := time.Duration(1<<attempt) * time.Second // 1s, 2s, 4s
		fmt.Fprintf(os.Stderr, "  retrying in %s (%v)\n", wait, err)
		time.Sleep(wait)
	}
	return "", err
}

func isRetryable(err error) bool {
	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		return apiErr.HTTPStatusCode == http.StatusTooManyRequests ||
			apiErr.HTTPStatusCode >= http.StatusInternalServerError
	}
	return false
}

func splitFile(path string) ([]string, func(), error) {
	tmp, err := os.MkdirTemp("", "transcriber-*")
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { os.RemoveAll(tmp) }

	ext := filepath.Ext(path)
	out := filepath.Join(tmp, "chunk_%03d"+ext)

	ffmpeg, _ := lookPath("ffmpeg")
	cmd := exec.Command(ffmpeg,
		"-i", path,
		"-f", "segment",
		"-segment_time", "600", // 10 min chunks
		"-c", "copy",
		"-loglevel", "error",
		out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("ffmpeg: %s", strings.TrimSpace(string(b)))
	}

	matches, _ := filepath.Glob(filepath.Join(tmp, "chunk_*"+ext))
	if len(matches) == 0 {
		cleanup()
		return nil, nil, fmt.Errorf("ffmpeg produced no output")
	}
	return matches, cleanup, nil
}

func whisperCall(apiKey, path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), whisperTimeout)
	defer cancel()
	client := openai.NewClient(apiKey)
	resp, err := client.CreateTranscription(ctx, openai.AudioRequest{
		Model:    openai.Whisper1,
		FilePath: path,
	})
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

func mergeChunks(apiKey string, parts []string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), mergeTimeout)
	defer cancel()
	client := openai.NewClient(apiKey)
	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: "You are given consecutive segments of an audio transcription, each separated by '---'. " +
					"Merge them into a single coherent transcript. Fix duplicate words or phrases at segment boundaries. " +
					"Preserve speaker changes and natural paragraph structure. Output only the merged transcript, nothing else.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: strings.Join(parts, "\n\n---\n\n"),
			},
		},
	})
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}
