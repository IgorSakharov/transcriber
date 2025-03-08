<?php

namespace App\Providers;

use App\Services\AwsService;
use Illuminate\Support\ServiceProvider;

class AwsServiceProvider extends ServiceProvider
{
    /**
     * Register services.
     */
    public function register(): void
    {
        $this->app->singleton(AwsService::class, function ($app) {
            return new AwsService();
        });
    }

    /**
     * Bootstrap services.
     */
    public function boot(): void
    {
        //
    }
}
