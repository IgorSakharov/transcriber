<template>
    <AppLayout title="Audio File Details">
        <template #header>
            <h2 class="font-semibold text-xl text-gray-800 leading-tight">
                Audio File Details
            </h2>
        </template>

        <div class="py-12">
            <div class="max-w-7xl mx-auto sm:px-6 lg:px-8">
                <div class="bg-white overflow-hidden shadow-xl sm:rounded-lg">
                    <div class="p-6">
                        <div class="mb-6">
                            <Link :href="route('audio-files.index')" class="text-indigo-600 hover:text-indigo-900">
                                ← Back to Audio Files
                            </Link>
                        </div>

                        <div class="space-y-6">
                            <div>
                                <h3 class="text-lg font-medium text-gray-900">File Information</h3>
                                <div class="mt-2 grid grid-cols-2 gap-4">
                                    <div>
                                        <p class="text-sm font-medium text-gray-500">File Name</p>
                                        <p class="mt-1 text-sm text-gray-900">{{ getFileName(audioFile.s3_input_key) }}</p>
                                    </div>
                                    <div>
                                        <p class="text-sm font-medium text-gray-500">Status</p>
                                        <p class="mt-1">
                                            <span :class="getStatusClasses(audioFile.status)" class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full">
                                                {{ capitalize(audioFile.status) }}
                                            </span>
                                        </p>
                                    </div>
                                    <div>
                                        <p class="text-sm font-medium text-gray-500">Uploaded</p>
                                        <p class="mt-1 text-sm text-gray-900">{{ formatDate(audioFile.created_at) }}</p>
                                    </div>
                                    <div>
                                        <p class="text-sm font-medium text-gray-500">Last Updated</p>
                                        <p class="mt-1 text-sm text-gray-900">{{ formatDate(audioFile.updated_at) }}</p>
                                    </div>
                                </div>
                            </div>

                            <div v-if="audioFile.status === 'transcribing'" class="bg-yellow-50 border-l-4 border-yellow-400 p-4">
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

                            <div v-if="audioFile.status === 'completed'" class="space-y-4">
                                <div>
                                    <h3 class="text-lg font-medium text-gray-900">Transcription</h3>
                                    <div class="mt-2 p-4 bg-gray-50 rounded-lg">
                                        <p class="text-sm text-gray-900 whitespace-pre-wrap">{{ audioFile.transcript_text }}</p>
                                    </div>
                                </div>
                                
                                <div class="flex justify-end">
                                    <button
                                        @click="copyTranscript"
                                        class="inline-flex items-center px-4 py-2 bg-white border border-gray-300 rounded-md font-semibold text-xs text-gray-700 uppercase tracking-widest shadow-sm hover:text-gray-500 focus:outline-none focus:border-blue-300 focus:ring focus:ring-blue-200 active:text-gray-800 active:bg-gray-50 disabled:opacity-25 transition"
                                    >
                                        <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
                                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3"></path>
                                        </svg>
                                        Copy Transcript
                                    </button>
                                </div>
                            </div>

                            <div v-if="audioFile.status === 'failed'" class="bg-red-50 border-l-4 border-red-400 p-4">
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
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </AppLayout>
</template>

<script setup>
import { onMounted } from 'vue';
import { Link, router } from '@inertiajs/vue3';
import AppLayout from '@/Layouts/AppLayout.vue';
import { format } from 'date-fns';

const props = defineProps({
    audioFile: {
        type: Object,
        required: true
    }
});

const getFileName = (path) => {
    return path.split('/').pop();
};

const capitalize = (str) => {
    return str.charAt(0).toUpperCase() + str.slice(1);
};

const formatDate = (date) => {
    return format(new Date(date), 'PPpp');
};

const getStatusClasses = (status) => {
    const classes = {
        completed: 'bg-green-100 text-green-800',
        failed: 'bg-red-100 text-red-800',
        transcribing: 'bg-yellow-100 text-yellow-800',
        uploaded: 'bg-gray-100 text-gray-800'
    };
    return classes[status] || classes.uploaded;
};

const copyTranscript = () => {
    navigator.clipboard.writeText(props.audioFile.transcript_text)
        .then(() => {
            // You could add a toast notification here
            console.log('Transcript copied to clipboard');
        })
        .catch(err => {
            console.error('Failed to copy transcript:', err);
        });
};

// Auto-refresh for transcribing status
let refreshInterval;

onMounted(() => {
    if (props.audioFile.status === 'transcribing') {
        refreshInterval = setInterval(() => {
            router.reload({ preserveScroll: true });
        }, 30000);
    }

    return () => {
        if (refreshInterval) {
            clearInterval(refreshInterval);
        }
    };
});
</script>
