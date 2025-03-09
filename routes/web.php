<?php

use App\Http\Controllers\AudioFileController;
use Illuminate\Foundation\Application;
use Illuminate\Support\Facades\Route;
use Inertia\Inertia;

Route::get('/', function () {
    return Inertia::render('Welcome', [
        'canLogin' => Route::has('login'),
        'canRegister' => Route::has('register'),
        'laravelVersion' => Application::VERSION,
        'phpVersion' => PHP_VERSION,
    ]);
});

Route::middleware([
    'auth:sanctum',
    config('jetstream.auth_session'),
    'verified',
])->group(function () {
    Route::get('/dashboard', function () {
        return Inertia::render('Dashboard');
    })->name('dashboard');

    Route::controller(AudioFileController::class)->group(function () {
        Route::get('/audio-files', 'index')->name('audio-files.index');
        Route::get('/audio-files/create', 'create')->name('audio-files.create');
        Route::post('/audio-files', 'store')->name('audio-files.store');
        Route::get('/audio-files/{audioFile}', 'show')->name('audio-files.show');
    });
});
