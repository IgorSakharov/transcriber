<?php

namespace Tests\Unit;

use App\Services\AwsService;
use PHPUnit\Framework\TestCase;

class AwsServiceTest extends TestCase
{
    /**
     * Helper to create an AwsService instance with fake clients.
     */
    private function makeService(object $transcribeClient, string $defaultLanguage): AwsService
    {
        $reflection = new \ReflectionClass(AwsService::class);
        $service = $reflection->newInstanceWithoutConstructor();

        foreach ([
            'bucket' => 'test-bucket',
            'defaultLanguage' => $defaultLanguage,
            'outputBucket' => 'test-bucket',
            's3Client' => new \stdClass(),
            'transcribeClient' => $transcribeClient,
        ] as $name => $value) {
            $property = $reflection->getProperty($name);
            $property->setAccessible(true);
            $property->setValue($service, $value);
        }

        return $service;
    }

    public function testUsesConfiguredLanguageAndNormalizesMediaFormat(): void
    {
        // Arrange
        $fakeClient = new class {
            public array $received;
            public function startTranscriptionJob(array $args)
            {
                $this->received = $args;
                return new class {
                    public function toArray() { return []; }
                };
            }
        };
        $service = $this->makeService($fakeClient, 'fr-FR');

        // Act
        $service->startTranscription('job-name', 's3://bucket/Audio.MP3');

        // Assert
        $this->assertSame('fr-FR', $fakeClient->received['LanguageCode']);
        $this->assertSame('mp3', $fakeClient->received['MediaFormat']);
        $this->assertSame('test-bucket', $fakeClient->received['OutputBucketName']);
    }

    public function testUsesProvidedLanguageWhenSupplied(): void
    {
        // Arrange
        $fakeClient = new class {
            public array $received;
            public function startTranscriptionJob(array $args)
            {
                $this->received = $args;
                return new class {
                    public function toArray() { return []; }
                };
            }
        };
        $service = $this->makeService($fakeClient, 'fr-FR');

        // Act
        $service->startTranscription('job-name', 's3://bucket/Audio.wav', 'es-ES');

        // Assert
        $this->assertSame('es-ES', $fakeClient->received['LanguageCode']);
        $this->assertSame('wav', $fakeClient->received['MediaFormat']);
        $this->assertSame('test-bucket', $fakeClient->received['OutputBucketName']);
    }
}
