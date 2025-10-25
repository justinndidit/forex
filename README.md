# 🌍 Country Currency & Exchange API


This is a RESTful API built with Go that fetches country and currency data from external sources, caches it in a PostgreSQL database, and provides a set of endpoints for querying the data. It includes functionalities for data refresh, filtering, and generating a summary image of the cached data.

This project is built with a clean architecture, using go-chi for routing, pgx for database interaction, and task for project automation.


## ✨ Features

    Data Aggregation: Fetches country data (name, capital, population, flag) from restcountries.com.

    Exchange Rate Integration: Fetches the latest USD exchange rates from open.er-api.com.

    Data Caching: Caches the combined and processed data in a PostgreSQL database.

    Dynamic GDP Calculation: Computes an estimated_gdp for each country on data refresh.

    Image Generation: Creates a summary image (cache/summary.png) of the top 5 countries by GDP after each refresh.

    RESTful Endpoints: Provides clean endpoints to read, filter, and delete country data.

## 🛠️ Tech Stack

    Backend: Go (Golang)

    Database: PostgreSQL 16

    Routing: go-chi/chi

    Database Driver: pgx

    Migrations: tern

    Task Runner: task (go-task)

    Live Reload: air

    Containerization: Docker & Docker Compose

## 🏁 Getting Started

Prerequisites

    Go (1.21 or later)

    Task

    Docker & Docker Compose

    Air (for live reload)

    Tern (for migrations)

1. Environment Setup

This project requires a .env file in the root directory. Copy the example file:

```bash

  cp .env.sample .env

```

Now, edit .env with your desired credentials:


2. Running with Docker (Recommended)

The easiest way to get the database and admin UI running is with Docker Compose.

    1. Start Services:

    ```bash

      docker compose up -d

    ````

    2. Access PGAdmin: You can access the database UI at http://localhost:5050 to monitor your database.

    3. Running Locally (Development)

    If you prefer to run the Go application directly on your host machine.

    1. Ensure DB is Running: Make sure you have a PostgreSQL database running (either via Docker from step 2 or a local install).

    2. Install Dependencies:

    ```bash

      task tidy

    ```

    3. Run Migrations: Apply the database schema.

    ```bash

      task migrations:up
    ```

    4. Run with Live Reload (Recommended):

    ```bash
      task run:dev
    ```

    5. Run (Once)

    ```bash
      task run
    ```

    6. The API will be available at http://localhost:8080 (or your configured port).


   ## 🗃️ Database Migrations

   Migrations are handled by tern.

    Create a new migration:

    ```bash

      task migrations:new name=your_migration_name

    ```
    This will create a new SQL file in ./internal/database/migrations.

    Apply all pending migrations:

    ```bash

      task migrations:up

    ```
    This will prompt for confirmation before running.


    📋 Available Tasks

This project uses task as a build tool. Here are the available commands:

    task run: Runs the Go application.

    task run:dev: Runs the application with air for live-reloading.

    task tidy: Formats Go files and tidies dependencies.

    task migrations:new name=...: Creates a new SQL migration file.

    task migrations:up: Applies all "up" migrations to the database.