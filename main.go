package main

import (
	"bufio"
	"encoding/json"
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
	var diarizeOutputPath string
	var speakerNames []string

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

	diarize := &cobra.Command{
		Use:          "diarize <file>",
		Short:        "Transcribe audio with speaker labels",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(speakerNames) > 4 {
				return fmt.Errorf("at most 4 speaker references allowed")
			}
			references := make([]speakerReference, len(speakerNames))
			for i, name := range speakerNames {
				reference, err := loadSpeakerReference(name)
				if err != nil {
					if os.IsNotExist(err) {
						return fmt.Errorf("speaker reference not found: %s", name)
					}
					return fmt.Errorf("loading speaker reference %s: %w", name, err)
				}
				references[i] = reference
			}
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

			result, err := diarizeFile(apiKey, args[0], references)
			if err != nil {
				return err
			}
			output, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return err
			}
			if diarizeOutputPath == "" {
				diarizeOutputPath = strings.TrimSuffix(args[0], filepath.Ext(args[0])) + ".diarized.json"
			}
			if err := os.WriteFile(diarizeOutputPath, append(output, '\n'), 0644); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Saved to", diarizeOutputPath)
			return nil
		},
	}
	diarize.Flags().StringVarP(&diarizeOutputPath, "output", "o", "", "write diarized transcript to file")
	diarize.Flags().StringArrayVar(&speakerNames, "speaker", nil, "saved speaker reference")
	root.AddCommand(diarize)

	speakers := &cobra.Command{
		Use:   "speakers",
		Short: "Manage speaker references",
	}
	speakers.AddCommand(&cobra.Command{
		Use:          "add <name> <sample-file>",
		Short:        "Save speaker reference audio",
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateInput(args[1]); err != nil {
				return fmt.Errorf("speaker sample: %w", err)
			}
			if err := saveSpeakerReference(args[0], args[1]); err != nil {
				return fmt.Errorf("saving speaker reference: %w", err)
			}
			fmt.Fprintln(os.Stderr, "Speaker saved:", args[0])
			return nil
		},
	})
	speakers.AddCommand(&cobra.Command{
		Use:          "list",
		Short:        "List speaker references",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := listSpeakerReferences()
			if err != nil {
				return err
			}
			for _, name := range names {
				fmt.Fprintln(cmd.OutOrStdout(), name)
			}
			return nil
		},
	})
	speakers.AddCommand(&cobra.Command{
		Use:          "remove <name>",
		Short:        "Remove speaker reference",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := removeSpeakerReference(args[0]); err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Speaker removed:", args[0])
			return nil
		},
	})
	speakers.AddCommand(&cobra.Command{
		Use:          "enroll <audio> <diarized-json>",
		Short:        "Save speaker references from diarized audio",
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateInput(args[0]); err != nil {
				return fmt.Errorf("audio: %w", err)
			}
			transcript, err := loadDiarizedTranscript(args[1])
			if err != nil {
				return fmt.Errorf("diarized transcript: %w", err)
			}
			candidates := unknownSpeakerRanges(transcript)
			if len(candidates) == 0 {
				return fmt.Errorf("no usable speaker candidates")
			}

			scanner := bufio.NewScanner(cmd.InOrStdin())
			fmt.Fprint(cmd.OutOrStdout(), "Update diarized JSON after naming? [y/N]: ")
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					return err
				}
				return fmt.Errorf("input required")
			}
			updateTranscript := strings.EqualFold(strings.TrimSpace(scanner.Text()), "y")
			usable := false
			for _, candidate := range candidates {
				if candidate.End-candidate.Start < 2 {
					continue
				}
				sample, cleanup, err := extractCandidateSample(args[0], candidate)
				if err != nil {
					return fmt.Errorf("extracting %s sample: %w", candidate.Speaker, err)
				}
				usable = true
				advance := false
				var playback *exec.Cmd
				stopPlayback := func() {
					if playback != nil {
						playback.Process.Kill()
						playback.Wait()
						playback = nil
					}
				}
				for !advance {
					fmt.Fprintf(cmd.OutOrStdout(), "%s [p]lay [r]eplay [x] stop [n]ame [s]kip [q]uit: ", candidate.Speaker)
					if !scanner.Scan() {
						stopPlayback()
						cleanup()
						if err := scanner.Err(); err != nil {
							return err
						}
						return fmt.Errorf("input required")
					}
					switch strings.TrimSpace(scanner.Text()) {
					case "p", "r":
						if playback != nil {
							stopPlayback()
						}
						var err error
						playback, err = playSample(sample)
						if err != nil {
							cleanup()
							return err
						}
					case "x":
						stopPlayback()
					case "n":
						stopPlayback()
						fmt.Fprint(cmd.OutOrStdout(), "Name: ")
						if !scanner.Scan() {
							cleanup()
							if err := scanner.Err(); err != nil {
								return err
							}
							return fmt.Errorf("speaker name required")
						}
						name := strings.TrimSpace(scanner.Text())
						if err := saveSpeakerReference(name, sample); err != nil {
							cleanup()
							return fmt.Errorf("saving speaker reference: %w", err)
						}
						if updateTranscript {
							raw, err := os.ReadFile(args[1])
							if err != nil {
								cleanup()
								return fmt.Errorf("reading diarized transcript: %w", err)
							}
							var document map[string]json.RawMessage
							if err := json.Unmarshal(raw, &document); err != nil {
								cleanup()
								return fmt.Errorf("parsing diarized transcript: %w", err)
							}
							var segments []map[string]json.RawMessage
							if err := json.Unmarshal(document["segments"], &segments); err != nil {
								cleanup()
								return fmt.Errorf("parsing diarized transcript segments: %w", err)
							}
							for _, segment := range segments {
								var speaker string
								if json.Unmarshal(segment["speaker"], &speaker) == nil && speaker == candidate.Speaker {
									segment["speaker"], _ = json.Marshal(name)
								}
							}
							document["segments"], err = json.Marshal(segments)
							if err != nil {
								cleanup()
								return err
							}
							output, err := json.MarshalIndent(document, "", "  ")
							if err != nil {
								cleanup()
								return err
							}
							if err := os.WriteFile(args[1], append(output, '\n'), 0644); err != nil {
								cleanup()
								return fmt.Errorf("updating diarized transcript: %w", err)
							}
						}
						fmt.Fprintln(os.Stderr, "Speaker saved:", name)
						cleanup()
						advance = true
					case "s":
						stopPlayback()
						cleanup()
						advance = true
					case "q":
						stopPlayback()
						cleanup()
						return nil
					default:
						stopPlayback()
						cleanup()
						return fmt.Errorf("invalid action %q — expected p, r, x, n, s, or q", strings.TrimSpace(scanner.Text()))
					}
				}
			}
			if !usable {
				return fmt.Errorf("no usable speaker candidates")
			}
			return nil
		},
	})
	root.AddCommand(speakers)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
