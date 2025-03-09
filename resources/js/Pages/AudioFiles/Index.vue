<template>
    <AppLayout title="Audio Files">
        <template #header>
            <h2 class="font-semibold text-xl text-gray-800 leading-tight">
                Audio Files
            </h2>
        </template>

        <div class="py-12">
            <div class="max-w-7xl mx-auto sm:px-6 lg:px-8">
                <div class="bg-white overflow-hidden shadow-xl sm:rounded-lg">
                    <div class="p-6">
                        <div class="flex justify-between items-center mb-6">
                            <h3 class="text-lg font-medium text-gray-900">Your Audio Files</h3>
                            <Link :href="route('audio-files.create')" 
                                  class="inline-flex items-center px-4 py-2 bg-gray-800 border border-transparent rounded-md font-semibold text-xs text-white uppercase tracking-widest hover:bg-gray-700">
                                Upload New File
                            </Link>
                        </div>

                        <div v-if="audioFiles.length === 0" class="text-gray-500 text-center py-4">
                            No audio files uploaded yet.
                        </div>
                        <div v-else class="overflow-x-auto">
                            <table class="min-w-full divide-y divide-gray-200">
                                <thead class="bg-gray-50">
                                    <tr>
                                        <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">File</th>
                                        <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                                        <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Created</th>
                                        <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                                    </tr>
                                </thead>
                                <tbody class="bg-white divide-y divide-gray-200">
                                    <tr v-for="file in audioFiles" :key="file.id">
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                                            {{ getFileName(file.s3_input_key) }}
                                        </td>
                                        <td class="px-6 py-4 whitespace-nowrap">
                                            <span :class="getStatusClasses(file.status)" class="px-2 inline-flex text-xs leading-5 font-semibold rounded-full">
                                                {{ capitalize(file.status) }}
                                            </span>
                                        </td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                            {{ formatDate(file.created_at) }}
                                        </td>
                                        <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                                            <Link :href="route('audio-files.show', file.id)" class="text-indigo-600 hover:text-indigo-900">
                                                View
                                            </Link>
                                        </td>
                                    </tr>
                                </tbody>
                            </table>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </AppLayout>
</template>

<script setup>
import { Link } from '@inertiajs/vue3';
import AppLayout from '@/Layouts/AppLayout.vue';
import { format, formatDistance } from 'date-fns';

const props = defineProps({
    audioFiles: {
        type: Array,
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
    return formatDistance(new Date(date), new Date(), { addSuffix: true });
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
</script>
