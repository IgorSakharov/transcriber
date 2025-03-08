<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;

class AudioFile extends Model
{
    use HasFactory;

    /**
     * The attributes that are mass assignable.
     *
     * @var array<string>
     */
    protected $fillable = [
        's3_input_key',
        'transcription_job_name',
        'status',
        'transcript_text',
    ];

    /**
     * The attributes that should be cast.
     *
     * @var array<string, string>
     */
    protected $casts = [
        'created_at' => 'datetime',
        'updated_at' => 'datetime',
    ];

    /**
     * The model's default values for attributes.
     *
     * @var array
     */
    protected $attributes = [
        'status' => 'uploaded',
    ];

    /**
     * Get the valid status values.
     *
     * @return array<string>
     */
    public static function getValidStatuses(): array
    {
        return [
            'uploaded',
            'transcribing',
            'completed',
            'failed'
        ];
    }

    /**
     * Get the user that owns the audio file.
     */
    public function user(): BelongsTo
    {
        return $this->belongsTo(User::class);
    }
}
