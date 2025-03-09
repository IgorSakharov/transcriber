<?php

namespace App\Console\Commands;

use App\Jobs\UpdateTranscriptionStatus;
use Illuminate\Console\Command;

class ProcessTranscriptionUpdates extends Command
{
    protected $signature = 'transcriptions:update';
    protected $description = 'Check and update status of ongoing transcriptions';

    public function handle(): void
    {
        UpdateTranscriptionStatus::dispatch();
        $this->info('Transcription status update job dispatched.');
    }
}
