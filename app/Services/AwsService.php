<?php

namespace App\Services;

use Aws\S3\S3Client;
use Aws\TranscribeService\TranscribeServiceClient;
use Illuminate\Support\Facades\Storage;
use RuntimeException;

class AwsService
{
    private S3Client $s3Client;
    private TranscribeServiceClient $transcribeClient;
    private string $bucket;

    private string $defaultLanguage;
    private string $outputBucket;

    public function __construct()
    {
        $this->bucket = config('filesystems.disks.s3.bucket');
        $this->defaultLanguage = config('services.aws.transcribe.language', 'en-US');
        $this->outputBucket = config('services.aws.transcribe.output_bucket', $this->bucket);
        
        $awsConfig = [
            'version' => 'latest',
            'region'  => config('filesystems.disks.s3.region'),
            'credentials' => [
                'key'    => config('filesystems.disks.s3.key'),
                'secret' => config('filesystems.disks.s3.secret'),
            ],
        ];

        if ($endpoint = config('filesystems.disks.s3.endpoint')) {
            $awsConfig['endpoint'] = $endpoint;
            $awsConfig['use_path_style_endpoint'] = config('filesystems.disks.s3.use_path_style_endpoint', false);
        }

        $this->s3Client = new S3Client($awsConfig);
        $this->transcribeClient = new TranscribeServiceClient($awsConfig);
    }

    /**
     * Upload a file to S3 and return the S3 URI
     */
    public function uploadFile(string $filePath, string $key): string
    {
        if (!Storage::disk('s3')->put($key, file_get_contents($filePath))) {
            throw new RuntimeException('Failed to upload file to S3');
        }

        return "s3://{$this->bucket}/{$key}";
    }

    /**
     * Start a transcription job
     */
    public function startTranscription(string $jobName, string $mediaUri, string $languageCode = 'en-US'): array
    {
        return $this->transcribeClient->startTranscriptionJob([
            'TranscriptionJobName' => $jobName,
            'Media' => [
                'MediaFileUri' => $mediaUri
            ],
            'MediaFormat' => pathinfo($mediaUri, PATHINFO_EXTENSION),
            'LanguageCode' => $languageCode,
        ]);
    }

    /**
     * Get transcription job status
     */
    public function getTranscriptionStatus(string $jobName): array
    {
        return $this->transcribeClient->getTranscriptionJob([
            'TranscriptionJobName' => $jobName
        ]);
    }

    /**
     * Get transcription result
     */
    public function getTranscriptionResult(string $resultUrl): string
    {
        $result = file_get_contents($resultUrl);
        $transcription = json_decode($result, true);
        
        return $transcription['results']['transcripts'][0]['transcript'] ?? '';
    }
}
