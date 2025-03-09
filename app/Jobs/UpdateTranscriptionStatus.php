<?php

namespace App\Jobs;

use App\Models\AudioFile;
use App\Services\AwsService;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Illuminate\Support\Facades\Log;
use RuntimeException;

class UpdateTranscriptionStatus implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    /**
     * The number of times the job may be attempted.
     *
     * @var int
     */
    public $tries = 3;

    /**
     * The number of seconds to wait before retrying the job.
     *
     * @var int
     */
    public $backoff = [10, 30, 60];

    /**
     * Execute the job.
     */
    public function handle(AwsService $awsService): void
    {
        Log::info('Starting transcription status update job');
        
        $transcribingFiles = AudioFile::where('status', 'transcribing')
            ->where('created_at', '>=', now()->subHours(24))
            ->get();

        if ($transcribingFiles->isEmpty()) {
            Log::info('No pending transcriptions found');
            return;
        }

        foreach ($transcribingFiles as $file) {
            try {
                $this->processTranscriptionFile($file, $awsService);
            } catch (RuntimeException $e) {
                Log::error("AWS Service error for file {$file->id}: {$e->getMessage()}");
                continue;
            } catch (\Exception $e) {
                Log::error("Unexpected error for file {$file->id}: {$e->getMessage()}");
                continue;
            }
        }

        Log::info('Completed transcription status update job');
    }

    /**
     * Process a single transcription file
     */
    private function processTranscriptionFile(AudioFile $file, AwsService $awsService): void
    {
        Log::info("Checking status for file: {$file->id}");
        
        $transcriptionStatus = $awsService->getTranscriptionStatus($file->transcription_job_name);
        $status = strtolower($transcriptionStatus['TranscriptionJob']['TranscriptionJobStatus']);

        switch ($status) {
            case 'completed':
                $this->handleCompletedTranscription($file, $transcriptionStatus, $awsService);
                break;
            
            case 'failed':
            case 'error':
                $this->handleFailedTranscription($file, $transcriptionStatus);
                break;
            
            case 'in_progress':
                Log::info("Transcription still in progress for file: {$file->id}");
                break;
            
            default:
                Log::warning("Unknown transcription status '{$status}' for file: {$file->id}");
        }
    }

    /**
     * Handle a completed transcription
     */
    private function handleCompletedTranscription(AudioFile $file, array $status, AwsService $awsService): void
    {
        $transcriptUrl = $status['TranscriptionJob']['Transcript']['TranscriptFileUri'];
        $transcriptText = $awsService->getTranscriptionResult($transcriptUrl);
        
        $file->update([
            'status' => 'completed',
            'transcript_text' => $transcriptText,
            'completed_at' => now()
        ]);

        Log::info("Successfully updated transcript for file: {$file->id}");
    }

    /**
     * Handle a failed transcription
     */
    private function handleFailedTranscription(AudioFile $file, array $status): void
    {
        $errorMessage = $status['TranscriptionJob']['FailureReason'] ?? 'Unknown error';
        
        $file->update([
            'status' => 'failed',
            'error_message' => $errorMessage
        ]);

        Log::error("Transcription failed for file {$file->id}: {$errorMessage}");
    }
}
