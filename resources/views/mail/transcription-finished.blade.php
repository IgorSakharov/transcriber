<!-- resources/views/mail/transcription-finished.blade.php -->
<x-mail::message>
# Audio Transcription Complete

Your audio file "{{ $fileName }}" has been successfully transcribed and is now ready for your review.

<x-mail::button :url="$url">
    View Transcription
</x-mail::button>

Thanks,<br>
{{ config('app.name') }}
</x-mail::message>