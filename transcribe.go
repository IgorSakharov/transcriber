package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
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
	if info.Size() < maxSize {
		fmt.Fprintln(os.Stderr, "Transcribing...")
		return whisperWithRetry(apiKey, path)
	}

	chunks, _, cleanup, err := splitFile(path)
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

func splitFile(path string) ([]string, []float64, func(), error) {
	tmp, err := os.MkdirTemp("", "transcriber-*")
	if err != nil {
		return nil, nil, nil, err
	}
	cleanup := func() { os.RemoveAll(tmp) }

	out := filepath.Join(tmp, "chunk_%03d.mp3")
	list := filepath.Join(tmp, "segments.csv")

	ffmpeg, err := lookPath("ffmpeg")
	if err != nil {
		cleanup()
		return nil, nil, nil, err
	}
	cmd := exec.Command(ffmpeg,
		"-i", path,
		"-map", "0:a:0",
		"-f", "segment",
		"-segment_time", "600",
		"-segment_list", list,
		"-segment_list_type", "csv",
		"-vn",
		"-c:a", "libmp3lame",
		"-b:a", "128k",
		"-loglevel", "error",
		out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		if output := strings.TrimSpace(string(b)); output != "" {
			return nil, nil, nil, fmt.Errorf("ffmpeg: %s", output)
		}
		return nil, nil, nil, fmt.Errorf("ffmpeg: %w", err)
	}

	matches, _ := filepath.Glob(filepath.Join(tmp, "chunk_*.mp3"))
	if len(matches) == 0 {
		cleanup()
		return nil, nil, nil, fmt.Errorf("ffmpeg produced no output")
	}
	sort.Slice(matches, func(i, j int) bool {
		left, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(filepath.Base(matches[i]), "chunk_"), ".mp3"))
		right, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(filepath.Base(matches[j]), "chunk_"), ".mp3"))
		return left < right
	})
	file, err := os.Open(list)
	if err != nil {
		cleanup()
		return nil, nil, nil, err
	}
	records, err := csv.NewReader(file).ReadAll()
	file.Close()
	if err != nil {
		cleanup()
		return nil, nil, nil, err
	}
	starts := make(map[string]float64, len(records))
	for _, record := range records {
		if len(record) < 2 {
			cleanup()
			return nil, nil, nil, fmt.Errorf("invalid ffmpeg segment list")
		}
		start, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			cleanup()
			return nil, nil, nil, err
		}
		starts[filepath.Base(record[0])] = start
	}
	offsets := make([]float64, len(matches))
	for i, match := range matches {
		start, ok := starts[filepath.Base(match)]
		if !ok {
			cleanup()
			return nil, nil, nil, fmt.Errorf("missing segment boundary for %s", filepath.Base(match))
		}
		offsets[i] = start
	}
	return matches, offsets, cleanup, nil
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

type diarizedSegment struct {
	Speaker string  `json:"speaker"`
	Start   float64 `json:"start"`
	End     float64 `json:"end"`
	Text    string  `json:"text"`
}

type diarizedTranscript struct {
	Segments      []diarizedSegment  `json:"segments"`
	KnownSpeakers map[string]struct{} `json:"known_speakers,omitempty"`
}

type speakerRange struct {
	Speaker string
	Start   float64
	End     float64
}

func extractCandidateSample(path string, candidate speakerRange) (string, func(), error) {
	duration := candidate.End - candidate.Start
	if duration < 2 {
		return "", nil, fmt.Errorf("sample must be at least 2 seconds")
	}
	if duration > 10 {
		duration = 10
	}
	ffmpeg, err := lookPath("ffmpeg")
	if err != nil {
		return "", nil, err
	}
	tmp, err := os.MkdirTemp("", "transcriber-sample-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(tmp) }
	sample := filepath.Join(tmp, "sample.mp3")
	cmd := exec.Command(ffmpeg, "-i", path, "-ss", strconv.FormatFloat(candidate.Start, 'f', -1, 64), "-t", strconv.FormatFloat(duration, 'f', -1, 64), "-vn", "-c:a", "libmp3lame", "-b:a", "128k", "-loglevel", "error", sample)
	if b, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		if output := strings.TrimSpace(string(b)); output != "" {
			return "", nil, fmt.Errorf("ffmpeg: %s", output)
		}
		return "", nil, fmt.Errorf("ffmpeg: %w", err)
	}
	return sample, cleanup, nil
}

func playSample(path string) (*exec.Cmd, error) {
	ffplay, err := lookPath("ffplay")
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(ffplay, "-nodisp", "-autoexit", "-loglevel", "error", path)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("ffplay: %w", err)
	}
	return cmd, nil
}

func loadDiarizedTranscript(path string) (diarizedTranscript, error) {
	file, err := os.Open(path)
	if err != nil {
		return diarizedTranscript{}, err
	}
	defer file.Close()

	var transcript diarizedTranscript
	if err := json.NewDecoder(file).Decode(&transcript); err != nil {
		return diarizedTranscript{}, err
	}
	return transcript, nil
}

func unknownSpeakerRanges(transcript diarizedTranscript) []speakerRange {
	ranges := make(map[string]speakerRange)
	var current speakerRange
	for _, segment := range transcript.Segments {
		if !isUnknownSpeaker(segment.Speaker, transcript.KnownSpeakers) {
			current = speakerRange{}
			continue
		}
		if current.Speaker == segment.Speaker {
			current.End = segment.End
		} else {
			current = speakerRange{Speaker: segment.Speaker, Start: segment.Start, End: segment.End}
		}
		if candidate, ok := ranges[current.Speaker]; !ok || current.End-current.Start > candidate.End-candidate.Start {
			ranges[current.Speaker] = current
		}
	}

	candidates := make([]speakerRange, 0, len(ranges))
	for _, candidate := range ranges {
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].End-candidates[i].Start > candidates[j].End-candidates[j].Start
	})
	return candidates
}

func isUnknownSpeaker(speaker string, knownSpeakers map[string]struct{}) bool {
	_, known := knownSpeakers[speaker]
	return !known
}

func diarizeFile(apiKey, path string, references []speakerReference) (diarizedTranscript, error) {
	if err := validateInput(path); err != nil {
		return diarizedTranscript{}, err
	}
	fmt.Fprintln(os.Stderr, "Diarization: preparing file...")
	knownSpeakers := make(map[string]struct{}, len(references))
	for i := range references {
		knownSpeakers[references[i].Name] = struct{}{}
		_, samplePath, err := speakerReferencePaths(references[i].Name, references[i].Sample)
		if err != nil {
			return diarizedTranscript{}, err
		}
		sample, err := os.ReadFile(samplePath)
		if err != nil {
			return diarizedTranscript{}, err
		}
		mediaType := mime.TypeByExtension(filepath.Ext(samplePath))
		if mediaType == "" {
			return diarizedTranscript{}, fmt.Errorf("unknown speaker sample type: %s", samplePath)
		}
		references[i].Sample = "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(sample)
	}
	info, err := os.Stat(path)
	if err != nil {
		return diarizedTranscript{}, err
	}
	if info.Size() < maxSize {
		fmt.Fprintln(os.Stderr, "  [1/1] processing")
		transcript, err := diarizeWithRetry(apiKey, path, references, "  [1/1]")
		transcript.KnownSpeakers = knownSpeakers
		return transcript, err
	}

	fmt.Fprintln(os.Stderr, "Diarization: splitting file...")
	chunks, offsets, cleanup, err := splitFile(path)
	if err != nil {
		return diarizedTranscript{}, err
	}
	defer cleanup()

	fmt.Fprintf(os.Stderr, "Diarization: %d chunks ready\n", len(chunks))
	parts := make([]diarizedTranscript, len(chunks))
	errs := make([]error, len(chunks))
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	for i, chunk := range chunks {
		wg.Add(1)
		go func(i int, chunk string) {
			defer wg.Done()
			fmt.Fprintf(os.Stderr, "  [%d/%d] queued\n", i+1, len(chunks))
			sem <- struct{}{}
			defer func() { <-sem }()
			label := fmt.Sprintf("  [%d/%d]", i+1, len(chunks))
			fmt.Fprintf(os.Stderr, "%s processing\n", label)
			parts[i], errs[i] = diarizeWithRetry(apiKey, chunk, references, label)
		}(i, chunk)
	}
	wg.Wait()

	result := diarizedTranscript{KnownSpeakers: knownSpeakers}
	for i, part := range parts {
		if errs[i] != nil {
			return diarizedTranscript{}, fmt.Errorf("chunk %d: %w", i+1, errs[i])
		}
		fmt.Fprintf(os.Stderr, "  [%d/%d] processing result\n", i+1, len(chunks))
		for _, segment := range part.Segments {
			if isUnknownSpeaker(segment.Speaker, knownSpeakers) {
				segment.Speaker = fmt.Sprintf("chunk_%03d:%s", i+1, segment.Speaker)
			}
			segment.Start += offsets[i]
			segment.End += offsets[i]
			result.Segments = append(result.Segments, segment)
		}
	}
	fmt.Fprintln(os.Stderr, "Diarization: assembling final transcript...")
	return result, nil
}

func diarizeWithRetry(apiKey, path string, references []speakerReference, label string) (diarizedTranscript, error) {
	var err error
	for attempt := range maxRetries {
		var transcript diarizedTranscript
		transcript, err = diarizeCall(apiKey, path, references, label)
		if err == nil {
			return transcript, nil
		}
		if !isRetryable(err) {
			return diarizedTranscript{}, err
		}
		wait := time.Duration(1<<attempt) * time.Second
		fmt.Fprintf(os.Stderr, "%s retrying in %s (%v)\n", label, wait, err)
		time.Sleep(wait)
	}
	return diarizedTranscript{}, err
}

func diarizeCall(apiKey, path string, references []speakerReference, label string) (diarizedTranscript, error) {
	file, err := os.Open(path)
	if err != nil {
		return diarizedTranscript{}, err
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return diarizedTranscript{}, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return diarizedTranscript{}, err
	}
	for key, value := range map[string]string{
		"model":             "gpt-4o-transcribe-diarize",
		"response_format":   "diarized_json",
		"chunking_strategy": "auto",
	} {
		if err := writer.WriteField(key, value); err != nil {
			return diarizedTranscript{}, err
		}
	}
	for _, reference := range references {
		if err := writer.WriteField("known_speaker_names[]", reference.Name); err != nil {
			return diarizedTranscript{}, err
		}
		if err := writer.WriteField("known_speaker_references[]", reference.Sample); err != nil {
			return diarizedTranscript{}, err
		}
	}
	if err := writer.Close(); err != nil {
		return diarizedTranscript{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), whisperTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/audio/transcriptions", &body)
	if err != nil {
		return diarizedTranscript{}, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	fmt.Fprintf(os.Stderr, "%s API request sent; waiting\n", label)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return diarizedTranscript{}, err
	}
	defer resp.Body.Close()
	fmt.Fprintf(os.Stderr, "%s API response received\n", label)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(resp.Body)
		return diarizedTranscript{}, &openai.APIError{HTTPStatusCode: resp.StatusCode, Message: strings.TrimSpace(string(message))}
	}
	var transcript diarizedTranscript
	if err := json.NewDecoder(resp.Body).Decode(&transcript); err != nil {
		return diarizedTranscript{}, err
	}
	return transcript, nil
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
