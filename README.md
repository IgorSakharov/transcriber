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

1. Add a **Filter Files** action to select audio files (wav, mp3, m4a, etc.)
2. Add a **Repeat with each item** action
3. Inside the loop, add a **Run Shell Script** action and paste:

```sh
export PATH="/opt/homebrew/bin:/usr/local/bin:$PATH"
transcriber "$1"
```

4. Set **Shell** to `zsh`, **Input** to `Repeat Item`, **Pass Input** to `as arguments`
5. The repeat results contain the transcripts — pipe into **Copy to Clipboard**, **Save File**, or any other action.

> **Note:** The API key must be set before using in Shortcuts. Run `transcriber set-key sk-...` once in Terminal.
