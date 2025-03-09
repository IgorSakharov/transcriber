<?php

namespace App\Policies;

use App\Models\AudioFile;
use App\Models\User;

class AudioFilePolicy
{
    /**
     * Determine whether the user can view the model.
     */
    public function view(User $user, AudioFile $audioFile): bool
    {
        return $user->id === $audioFile->user_id;
    }

    /**
     * Determine whether the user can create models.
     */
    public function create(User $user): bool
    {
        return true;
    }

    /**
     * Determine whether the user can delete the model.
     */
    public function delete(User $user, AudioFile $audioFile): bool
    {
        return $user->id === $audioFile->user_id;
    }
}
