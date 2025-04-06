<?php

namespace App\Http\Controllers;

use App\Models\AudioFile;
use App\Models\User;
use App\Services\AwsService;
use Illuminate\Http\Request;
use Illuminate\Support\Str;
use Illuminate\Support\Facades\Auth;
use Inertia\Inertia;

class AudioFileController extends Controller
{
    private AwsService $awsService;

    public function __construct(AwsService $awsService)
    {
        $this->awsService = $awsService;
    }

    /**
     * Display a listing of audio files.
     */
    public function index()
    {
        /** @var User $user */
        $user = Auth::user();
        $audioFiles = Auth::user()->audioFiles()->latest()->get();
        return Inertia::render('AudioFiles/Index', [
            'audioFiles' => $audioFiles
        ]);
    }

    /**
     * Show the form for creating a new audio file.
     */
    public function create()
    {
        return Inertia::render('AudioFiles/Create');
    }

    /**
     * Store a newly uploaded audio file.
     */
    public function store(Request $request)
    {
        $request->validate([
            'audio_file' => ['required', 'file', 'mimes:mp3,wav,m4a,flac', 'max:100000']
        ]);

        $file = $request->file('audio_file');
        $extension = $file->getClientOriginalExtension();

        // Generate a unique key for S3
        $key = sprintf(
            'audio/%s/%s.%s',
            Auth::id(),
            Str::uuid(),
            $extension
        );

        try {
            // Upload to S3
            $s3Uri = $this->awsService->uploadFile($file->path(), $key);

            // Create database record
            $audioFile = Auth::user()->audioFiles()->create([
                's3_input_key' => $key,
                'status' => 'uploaded'
            ]);

            // Start transcription
            $jobName = 'transcription-' . Str::uuid();
            $this->awsService->startTranscription($jobName, $s3Uri);

            // Update the job name
            $audioFile->update(['transcription_job_name' => $jobName, 'status' => 'transcribing']);

            return redirect()->route('audio-files.index')->with([
                'message' => 'Audio file uploaded successfully and transcription started.',
                'type' => 'success'
            ]);

        } catch (\Exception $e) {
            return back()
                ->withInput()
                ->withErrors(['audio_file' => 'Failed to upload audio file: ' . $e->getMessage()]);
        }
    }

    /**
     * Display the specified audio file.
     */
    public function show(AudioFile $audioFile)
    {
//        $this->authorize('view', $audioFile);


        if ($audioFile->status === 'transcribing') {
            try {
                $status = $this->awsService->getTranscriptionStatus($audioFile->transcription_job_name);

                if ($status['TranscriptionJob']['TranscriptionJobStatus'] === 'COMPLETED') {
                    $transcript = $this->awsService->getTranscriptionResult(
                        $status['TranscriptionJob']['Transcript']['TranscriptFileUri']
                    );

                    $audioFile->update([
                        'status' => 'completed',
                        'transcript_text' => $transcript
                    ]);
                } elseif ($status['TranscriptionJob']['TranscriptionJobStatus'] === 'FAILED') {
                    $audioFile->update(['status' => 'failed']);
                }
            } catch (\Exception $e) {
                // Log the error but don't throw it to the user
                \Log::error('Failed to check transcription status: ' . $e->getMessage());
            }
        }

        return Inertia::render('AudioFiles/Show', [
            'audioFile' => $audioFile
        ]);
    }
}
