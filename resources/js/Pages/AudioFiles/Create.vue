<template>
    <AppLayout title="Upload Audio File">
        <template #header>
            <h2 class="font-semibold text-xl text-gray-800 leading-tight">
                Upload Audio File
            </h2>
        </template>

        <div class="py-12">
            <div class="max-w-7xl mx-auto sm:px-6 lg:px-8">
                <div class="bg-white overflow-hidden shadow-xl sm:rounded-lg">
                    <div class="p-6">
                        <form @submit.prevent="submit" class="space-y-6">
                            <div>
                                <InputLabel for="audio_file" value="Audio File" />
                                <input
                                    id="audio_file"
                                    ref="audioFile"
                                    type="file"
                                    @input="form.audio_file = $event.target.files[0]"
                                    accept=".mp3,.wav,.m4a,.flac"
                                    class="mt-1 block w-full"
                                    required
                                />
                                <p class="mt-2 text-sm text-gray-500">
                                    Supported formats: MP3, WAV, M4A, FLAC. Maximum size: 50MB
                                </p>
                                <InputError :message="form.errors.audio_file" class="mt-2" />
                            </div>

                            <div class="flex items-center justify-end mt-4">
                                <Link
                                    :href="route('audio-files.index')"
                                    class="inline-flex items-center px-4 py-2 bg-gray-300 border border-transparent rounded-md font-semibold text-xs text-gray-700 uppercase tracking-widest hover:bg-gray-400 active:bg-gray-400 focus:outline-none focus:border-gray-900 focus:ring focus:ring-gray-300 disabled:opacity-25 transition mr-4"
                                >
                                    Cancel
                                </Link>
                                <PrimaryButton
                                    class="ml-4"
                                    :class="{ 'opacity-25': form.processing }"
                                    :disabled="form.processing"
                                >
                                    <div v-if="form.processing" class="mr-2">
                                        <svg class="animate-spin h-4 w-4 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                                            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                                            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                                        </svg>
                                    </div>
                                    Upload
                                </PrimaryButton>
                            </div>
                        </form>
                    </div>
                </div>
            </div>
        </div>
    </AppLayout>
</template>

<script setup>
import { ref } from 'vue';
import { Link, useForm } from '@inertiajs/vue3';
import AppLayout from '@/Layouts/AppLayout.vue';
import InputLabel from '@/Components/InputLabel.vue';
import InputError from '@/Components/InputError.vue';
import PrimaryButton from '@/Components/PrimaryButton.vue';

const form = useForm({
    audio_file: null,
});

const submit = () => {
    form.post(route('audio-files.store'), {
        preserveScroll: true,
        onSuccess: () => {
            form.reset();
        },
    });
};
</script>
