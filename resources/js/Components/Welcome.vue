
<template>
    <div>
        <div class="p-6 lg:p-8 bg-white dark:bg-gray-800 dark:bg-gradient-to-bl dark:from-gray-700/50 dark:via-transparent border-b border-gray-200 dark:border-gray-700">
            <h1 class="mt-8 text-2xl font-medium text-gray-900 dark:text-white">
                Welcome to Transcriber application!
            </h1>

            <p class="mt-6 text-gray-500 dark:text-gray-400 leading-relaxed">
                Tha application does not have the dashboard page yet :) Stay tuned for new updates!
                Check out our main feature for now.
            </p>
            <ResponsiveNavLink :href="route('audio-files.index')" :active="route().current('audio-files.*')">
                Audio Files
            </ResponsiveNavLink>
        </div>
    </div>
</template>
<script setup lang="ts">
import ResponsiveNavLink from "@/Components/ResponsiveNavLink.vue";
</script>
