package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// ponytail: also probe common Homebrew paths — Shortcuts/launchd don't inherit shell PATH
var extraPaths = []string{"/opt/homebrew/bin", "/usr/local/bin"}

func lookPath(bin string) (string, error) {
	if p, err := exec.LookPath(bin); err == nil {
		return p, nil
	}
	for _, dir := range extraPaths {
		p := dir + "/" + bin
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("%s not found", bin)
}

func checkDeps() error {
	if _, err := lookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found — install with: brew install ffmpeg")
	}
	return nil
}

func main() {
	var outputPath string

	root := &cobra.Command{
		Use:          "transcriber <file>",
		Short:        "Transcribe audio files via OpenAI Whisper",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkDeps(); err != nil {
				return err
			}
			apiKey, err := getAPIKey()
			if err != nil {
				return err
			}

			if err := acquireLock(args[0]); err != nil {
				return err
			}
			defer releaseLock(args[0])

			result, err := transcribeFile(apiKey, args[0])
			if err != nil {
				return err
			}

			if outputPath == "" {
				outputPath = strings.TrimSuffix(args[0], filepath.Ext(args[0])) + ".txt"
			}
			if err := os.WriteFile(outputPath, []byte(result+"\n"), 0644); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Saved to", outputPath)
			return nil
		},
	}

	root.Flags().StringVarP(&outputPath, "output", "o", "", "write transcript to file instead of stdout")

	root.AddCommand(&cobra.Command{
		Use:          "set-key <api-key>",
		Short:        "Store OpenAI API key",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := saveAPIKey(args[0]); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Key saved.")
			return nil
		},
	})

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
