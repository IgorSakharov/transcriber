<x-app-layout>
    <x-slot name="header">
        <h2 class="font-semibold text-xl text-gray-800 leading-tight">
            {{ __('Audio File Details') }}
        </h2>
    </x-slot>

    <div class="py-12">
        <div class="max-w-7xl mx-auto sm:px-6 lg:px-8">
            <div class="bg-white overflow-hidden shadow-xl sm:rounded-lg">
                <div class="p-6">
                    <div class="mb-6">
                        <a href="{{ route('audio-files.index') }}" class="text-indigo-600 hover:text-indigo-900">
                            ← Back to Audio Files
                        </a>
                    </div>

                    <div class="space-y-6">
                        <div>
                            <h3 class="text-lg font-medium text-gray-900">File Information</h3>
                            <div class="mt-2 grid grid-cols-2 gap-4">
                                <div>
                                    <p class="text-sm font-medium text-gray-500">File Name</p>
                                    <p class="mt-1 text-sm text-gray-900">{{ basename($audioFile->s3_input_key) }}</p>
                                </div>
                                <div>
                                    <p class="text-sm font-medium text-gray-500">Status</p>
                                    <p class="mt-1">
                                        <span class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full 
                                            @if($audioFile->status === 'completed') bg-green-100 text-green-800
                                            @elseif($audioFile->status === 'failed') bg-red-100 text-red-800
                                            @elseif($audioFile->status === 'transcribing') bg-yellow-100 text-yellow-800
                                            @else bg-gray-100 text-gray-800 @endif">
                                            {{ ucfirst($audioFile->status) }}
                                        </span>
                                    </p>
                                </div>
                                <div>
                                    <p class="text-sm font-medium text-gray-500">Uploaded</p>
                                    <p class="mt-1 text-sm text-gray-900">{{ $audioFile->created_at->format('F j, Y g:i A') }}</p>
                                </div>
                                <div>
                                    <p class="text-sm font-medium text-gray-500">Last Updated</p>
                                    <p class="mt-1 text-sm text-gray-900">{{ $audioFile->updated_at->format('F j, Y g:i A') }}</p>
                                </div>
                            </div>
                        </div>

                        @if($audioFile->status === 'transcribing')
                            <div class="bg-yellow-50 border-l-4 border-yellow-400 p-4">
                                <div class="flex">
                                    <div class="flex-shrink-0">
                                        <svg class="h-5 w-5 text-yellow-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                                            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z" clip-rule="evenodd" />
                                        </svg>
                                    </div>
                                    <div class="ml-3">
                                        <p class="text-sm text-yellow-700">
                                            Transcription is in progress. This page will automatically refresh every 30 seconds.
                                        </p>
                                    </div>
                                </div>
                            </div>

                            @push('scripts')
                            <script>
                                setTimeout(function() {
                                    window.location.reload();
                                }, 30000);
                            </script>
                            @endpush
                        @endif

                        @if($audioFile->status === 'completed')
                            <div>
                                <h3 class="text-lg font-medium text-gray-900">Transcription</h3>
                                <div class="mt-2 p-4 bg-gray-50 rounded-lg">
                                    <p class="text-sm text-gray-900 whitespace-pre-wrap">{{ $audioFile->transcript_text }}</p>
                                </div>
                            </div>
                        @endif

                        @if($audioFile->status === 'failed')
                            <div class="bg-red-50 border-l-4 border-red-400 p-4">
                                <div class="flex">
                                    <div class="flex-shrink-0">
                                        <svg class="h-5 w-5 text-red-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor">
                                            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
                                        </svg>
                                    </div>
                                    <div class="ml-3">
                                        <p class="text-sm text-red-700">
                                            Transcription failed. Please try uploading the file again.
                                        </p>
                                    </div>
                                </div>
                            </div>
                        @endif
                    </div>
                </div>
            </div>
        </div>
    </div>
</x-app-layout>
