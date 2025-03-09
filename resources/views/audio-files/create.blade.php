<x-app-layout>
    <x-slot name="header">
        <h2 class="font-semibold text-xl text-gray-800 leading-tight">
            {{ __('Upload Audio File') }}
        </h2>
    </x-slot>

    <div class="py-12">
        <div class="max-w-7xl mx-auto sm:px-6 lg:px-8">
            <div class="bg-white overflow-hidden shadow-xl sm:rounded-lg">
                <div class="p-6">
                    <form action="{{ route('audio-files.store') }}" method="POST" enctype="multipart/form-data" class="space-y-6">
                        @csrf

                        <div>
                            <x-label for="audio_file" value="{{ __('Audio File') }}" />
                            <input id="audio_file" 
                                   type="file" 
                                   name="audio_file"
                                   accept=".mp3,.wav,.m4a,.flac"
                                   class="mt-1 block w-full" 
                                   required />
                            <p class="mt-2 text-sm text-gray-500">
                                Supported formats: MP3, WAV, M4A, FLAC. Maximum size: 50MB
                            </p>
                            <x-input-error for="audio_file" class="mt-2" />
                        </div>

                        <div class="flex items-center justify-end mt-4">
                            <x-button>
                                {{ __('Upload') }}
                            </x-button>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    </div>
</x-app-layout>
