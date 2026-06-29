# transcriber

CLI tool to transcribe audio files via OpenAI Whisper. Handles large files by chunking and merging.

## Install

```sh
brew tap IgorSakharov/tap
brew install transcriber
```

Requires [ffmpeg](https://ffmpeg.org/) for files over 24MB:

```sh
brew install ffmpeg
```

## Usage

```sh
# print transcript to stdout
transcriber audio.m4a

# save to file
transcriber audio.m4a -o transcript.txt

# store API key (one-time setup)
transcriber set-key sk-...
```

On first run without a saved key, you will be prompted to enter it. The key is stored in `~/.transcriber_key`.

## Apple Shortcuts

1. Add a **Run Shell Script** action
2. Set **Shell** to `/bin/zsh`
3. Set **Input** to the audio file from a previous action (e.g. "Get File", "Record Audio")
4. Paste this script:

```sh
#!/bin/zsh
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"
transcriber "$1"
```

5. The action output contains the transcript — pipe it into **Copy to Clipboard**, **Create Text File**, or any other action.

> **Note:** The API key must be set before using in Shortcuts. Run `transcriber set-key sk-...` once in Terminal.
