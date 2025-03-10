<?php

namespace App\Console\Commands;

use App\Models\AudioFile;
use App\Models\User;
use App\Notifications\TranscriptionFinished;
use Illuminate\Console\Command;

class TestTranscriptionNotification extends Command
{
    protected $signature = 'notification:test-transcription {file}';
    protected $description = 'Test transcription completion notification';

    public function handle()
    {
        $fileId = $this->argument('file');
        $audioFile = AudioFile::findOrFail($fileId);
        $user = $audioFile->user;

        $user->notify(new TranscriptionFinished(
            $audioFile->transcription_job_name,
            route('audio-files.show', ['user' => $user, 'audio_file' => $audioFile])
        ));

        $this->info("Notification sent to user {$user->email}");
    }
}
