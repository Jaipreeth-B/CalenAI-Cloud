# CalenAI

[![Go Version](https://img.shields.io/badge/Go-1.25.9-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![Gin Framework](https://img.shields.io/badge/Gin-v1.12.0-008ECF?style=flat-square&logo=go)](https://gin-gonic.com/)
[![Svelte](https://img.shields.io/badge/Svelte-v5.55.5-FF3E00?style=flat-square&logo=svelte)](https://svelte.dev/)
[![Vite](https://img.shields.io/badge/Vite-v8.0.12-646CFF?style=flat-square&logo=vite)](https://vitejs.dev/)
[![TailwindCSS](https://img.shields.io/badge/TailwindCSS-v4.3.0-38B2AC?style=flat-square&logo=tailwind-css)](https://tailwindcss.com/)
[![OpenRouter](https://img.shields.io/badge/LLM-Nemotron--3--550B-76B900?style=flat-square&logo=nvidia)](https://openrouter.ai/nvidia/nemotron-3-ultra-550b-a55b:free)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square)](LICENSE)

An intelligent, full-stack AI calendar and task management application combining a reactive Svelte 5 frontend with a high-performance Go backend, multi-turn AI tool calling, and dual database support (Cloud PostgreSQL & SQLite).

---

## Overview

Managing busy daily routines through traditional calendar forms is tedious and prone to scheduling mistakes. **CalenAI** transforms task and agenda planning into a natural, conversational experience.

Users can speak naturally to schedule meetings, batch-create repeating tasks, reschedule appointments, mark items completed, or query upcoming obligations. Rather than generating text blindly, the assistant leverages native LLM tool calling (function calling) powered by NVIDIA's Nemotron model against a structured backend API to ensure that events, conflicts, and dates are checked and stored accurately.

### Who It Is For
- Professionals seeking a friction-free conversational calendar assistant.
- Developers looking for a production-ready reference architecture integrating Go (Gin + GORM) with Svelte 5 and OpenRouter AI function calling.

---

## Key Features

- **Conversational Task & Event Scheduling**: Create, reschedule, complete, and delete events using plain English.
- **Powered by NVIDIA Nemotron-3 550B**: Employs `nvidia/nemotron-3-ultra-550b-a55b:free` via OpenRouter for high-precision tool calling, multi-step logic, and natural conversation.
- **Authoritative Client Time Awareness**: The frontend transmits the user's exact device timestamp (`X-Client-Time`) and IANA timezone (`X-Timezone`) on every request, eliminating timezone shifts between client and cloud.
- **Strict Anti-Hallucination Time Enforcement**: Enforces explicit clock times (e.g. `3:00 PM`, `14:30`) and rejects ambiguous expressions (`morning`, `later`) by prompting for clarification.
- **Single-Call Batch Creation (`add_tasks`)**: Allows creating recurring or multi-day tasks (e.g. daily standups for an entire month) in a single request instead of entering recursive agent loops.
- **Pre-Creation Overlap Detection & Force Resolution**: Validates new tasks against existing bookings to avoid conflicts, returning an `OVERLAP_WARNING` with details and supporting an explicit `force` override if the user chooses to "keep both".
- **Multi-Turn Persistent Chat Sessions**: Full conversation history stored in sessions, supporting multi-step reasoning, conversation switching, and request cancellation via `AbortController`.
- **Dual-Mode Database Architecture**: Works seamlessly with **Cloud PostgreSQL** (e.g. Supabase connection poolers using `PreferSimpleProtocol`) or falls back automatically to local **SQLite** (`calenai.db`).
- **Reactive UI**: Built with modern **Svelte 5** and **Tailwind CSS v4**, featuring an interactive monthly calendar, day-view schedule filters, and conversational sidebars.

---

## How It Works

```mermaid
sequenceDiagram
    autonumber
    actor User
    participant Frontend as Svelte 5 Client
    participant Backend as Go (Gin Engine)
    participant DB as Cloud DB (PostgreSQL) / SQLite
    participant LLM as OpenRouter API (Nemotron-3 550B)

    User->>Frontend: Sends request (e.g. "Schedule gym at 7 AM tomorrow")
    Frontend->>Backend: POST /api/chat/:session_id (Headers: X-Timezone, X-Client-Time)
    Backend->>Backend: Inject authoritative current time & prompt constraints
    Backend->>LLM: POST /chat/completions (Tools: add_task, add_tasks, find_tasks...)
    LLM-->>Backend: Tool call request (e.g. add_task with parameters)
    Backend->>DB: Query overlaps & insert Task
    DB-->>Backend: Confirmation / Task Record
    Backend->>LLM: Return Tool Result
    LLM-->>Backend: Final assistant natural language response
    Backend->>DB: Persist messages to Session history
    Backend-->>Frontend: HTTP 200 { response: "..." }
    Frontend->>Backend: GET /api/tasks (Refreshes reactive UI)
    Backend-->>Frontend: Updated task list
    Frontend-->>User: Displays assistant reply & updates Calendar
```

---

## Tech Stack

| Layer | Technology | Version | Purpose |
| :--- | :--- | :--- | :--- |
| **Backend Language** | [Go](https://go.dev/) | `1.25.9` | Primary backend runtime |
| **Web Framework** | [Gin](https://github.com/gin-gonic/gin) | `v1.12.0` | High-performance HTTP router and middleware |
| **ORM** | [GORM](https://gorm.io/) | `v1.31.2` | Database object-relational mapping & schema auto-migrations |
| **PostgreSQL Driver** | `gorm.io/driver/postgres` | `v1.6.3` | Cloud PostgreSQL driver (Supabase / PgBouncer pooler support) |
| **SQLite Driver** | `github.com/glebarez/sqlite` | `v1.11.0` | Pure-Go embedded SQLite driver for offline local fallback |
| **Frontend Framework** | [Svelte](https://svelte.dev/) | `v5.55.5` | Reactive component framework |
| **Build Tool** | [Vite](https://vitejs.dev/) | `v8.0.12` | Frontend development server and production bundler |
| **Styling** | [Tailwind CSS](https://tailwindcss.com/) | `v4.3.0` | Utility-first CSS framework |
| **AI Inference** | [OpenRouter](https://openrouter.ai/) | Cloud API | Function-calling LLM provider (`nvidia/nemotron-3-ultra-550b-a55b:free`) |

---

## Project Structure

```text
CalenAI/
├── .gitignore               # Root ignore rules for build artifacts, databases, and .env files
├── .env.example             # Template environment variables for root reference
├── LICENSE                  # MIT License
├── README.md                # Project documentation
├── backend/
│   ├── .env                 # Local backend secrets (git-ignored)
│   ├── .env.example         # Template environment file for backend
│   ├── go.mod               # Go module declaration and direct dependencies
│   ├── go.sum               # Cryptographic checksums of Go dependencies
│   ├── main.go              # Complete backend server (routes, GORM models, OpenRouter tool caller)
│   └── ToDo.md              # Engineering roadmap and reliability tracking
└── frontend/
    ├── .env                 # Local frontend secrets (git-ignored)
    ├── .env.example         # Template environment file for frontend
    ├── .gitignore           # Frontend-specific ignore file
    ├── index.html           # HTML entry point
    ├── jsconfig.json        # JavaScript compiler & path configuration
    ├── package.json         # NPM scripts and frontend dependencies
    ├── svelte.config.js     # Svelte preprocessor configuration
    ├── vite.config.js       # Vite configuration with Tailwind CSS plugin
    └── src/
        ├── App.svelte       # Main application layout wrapper
        ├── app.css          # Global Tailwind directives
        ├── main.js          # Svelte 5 application mounting point
        ├── assets/          # Static frontend assets
        └── lib/
            ├── api.js       # Svelte stores, API clients, and dynamic header generator
            ├── Calendar.svelte # Interactive monthly calendar component
            ├── Chat.svelte     # AI chat panel with streaming cancellation support
            ├── Sidebar.svelte  # Chat sessions list and navigation
            └── TaskList.svelte # Agenda view, completion toggles, and deletion controls
```

---

## Prerequisites

Before running the project locally or deploying, ensure you have:

- **Go**: Version `1.22+` (configured with `1.25.9`).
- **Node.js**: Version `18.0.0+` or `20.0.0+` with `npm` (or `bun` / `pnpm`).
- **OpenRouter API Key**: A valid key from [OpenRouter](https://openrouter.ai/).
- **Database (Choose One)**:
  - **Local**: Nothing needed (SQLite creates `calenai.db` automatically).
  - **Cloud**: A PostgreSQL database connection string (e.g., from [Supabase](https://supabase.com), [Neon](https://neon.tech), or AWS RDS).

---

## Environment Variables

### Backend (`backend/.env`)

| Variable Name | Description | Required? | Example Placeholder |
| :--- | :--- | :--- | :--- |
| `OPENROUTER_API_KEY` | Bearer token for OpenRouter chat completion API | **Yes** | `your_openrouter_api_key_here` |
| `OPENROUTER_MODEL` | Target LLM identifier on OpenRouter | Optional (default: `nvidia/nemotron-3-ultra-550b-a55b:free`) | `nvidia/nemotron-3-ultra-550b-a55b:free` |
| `OPENROUTER_BASE_URL` | Base endpoint for OpenRouter completions | Optional | `https://openrouter.ai/api/v1/chat/completions` |
| `PORT` | Local network port for the Go server | Optional (default: `9090`) | `9090` |
| `GIN_MODE` | Runtime mode for Gin (`debug` or `release`) | Optional (default: `release`) | `release` |
| `DATABASE_URL` | Cloud PostgreSQL connection string (DSN or URI) | Optional (falls back to SQLite) | `postgresql://user:pass@host:6543/postgres?sslmode=require` |
| `DATABASE_PATH` | Local SQLite database filename | Optional (default: `calenai.db`) | `calenai.db` |

### Frontend (`frontend/.env`)

| Variable Name | Description | Required? | Example Placeholder |
| :--- | :--- | :--- | :--- |
| `VITE_API_URL` | Base URL pointing to the backend API | **Yes** | `http://localhost:9090/api` |

---

## Installation & Local Setup

### 1. Clone Repository
```bash
git clone https://github.com/jaipreeth-b/CalenAI.git
cd CalenAI
```

### 2. Configure Backend
```bash
cd backend
cp .env.example .env
```
Open `backend/.env` and insert your `OPENROUTER_API_KEY` (and optional `DATABASE_URL`).

Install backend dependencies:
```bash
go mod download
```

### 3. Configure Frontend
```bash
cd ../frontend
cp .env.example .env
```
Install frontend dependencies:
```bash
npm install
```

---

## Running the Project

### Running in Development

Run both services in separate terminal windows:

**Terminal 1 (Backend):**
```bash
cd backend
go run main.go
```
*Backend initializes on `http://localhost:9090`.*

**Terminal 2 (Frontend):**
```bash
cd frontend
npm run dev
```
*Frontend runs on `http://localhost:5173`.*

Open `http://localhost:5173` in your browser.

### Running in Production

**Backend:**
```bash
cd backend
go build -o calenai-server .
./calenai-server
```

**Frontend:**
```bash
cd frontend
npm run build
npm run preview
```

---

## Accuracy & System Verification Tests

CalenAI's AI decision engine has been tested across 10 verification scenarios using `nvidia/nemotron-3-ultra-550b-a55b:free`. **All tests have passed successfully (100% pass rate).**

### 1. General Conversation & Tool Restraint (No Tools Allowed)
*Goal: Ensure the AI does not trigger tools unnecessarily for basic chat.*

| # | Prompt to Test | Expected Behavior | Status |
| :--- | :--- | :--- | :---: |
| **1.1** | `"Hello! Who are you and what can you help me with?"` | Friendly greeting; 0 tool calls made. | **PASSED** |
| **1.2** | `"What is the capital of France?"` | Answers directly; 0 tool calls made. | **PASSED** |

### 2. Authoritative Time & Date Accuracy
*Goal: Ensure it uses the backend header time and parses relative dates properly.*

| # | Prompt to Test | Expected Behavior | Status |
| :--- | :--- | :--- | :---: |
| **2.1** | `"What is today's date and what time is it right now?"` | Answers using authoritative server values; no hallucinated date/year. | **PASSED** |
| **2.2** | `"What day of the week is tomorrow?"` | Calculates the exact day from the authoritative date. | **PASSED** |

### 3. Strict Time Enforcement (Anti-Hallucination)
*Goal: System prompt explicitly forbids vague words ('morning', 'evening') and requires asking for an exact clock time.*

| # | Prompt to Test | Expected Behavior | Status |
| :--- | :--- | :--- | :---: |
| **3.1** | `"Schedule a team standup tomorrow morning."` | Must NOT call `add_task`. Asks: *"What time tomorrow morning would you like to schedule this?"* | **PASSED** |
| **3.2** | `"Add a dentist appointment for Friday."` | Must NOT guess a time. Asks for a specific clock time. | **PASSED** |

### 4. Single Task Creation & Default Duration
*Goal: Tests `add_task` with specific clock times and duration calculation.*

| # | Prompt to Test | Expected Behavior | Status |
| :--- | :--- | :--- | :---: |
| **4.1** | `"Schedule a project review tomorrow at 3 PM."` | Calls `add_task` with computed start date-time; default duration of 1 hour (3 PM - 4 PM). | **PASSED** |
| **4.2** | `"Add an interview on November 10 at 10:30 AM for 2 hours."` | Calls `add_task` with start 10:30 and explicit end 12:30. | **PASSED** |

### 5. Multi-Day & Batch Creation (`add_tasks` in a Single Go)
*Goal: Tests recurring batch optimization in a single tool call.*

| # | Prompt to Test | Expected Behavior | Status |
| :--- | :--- | :--- | :---: |
| **5.1** | `"Add gym workout at 7:00 AM every day for the next 7 days."` | Calls `add_tasks` once containing an array of all 7 dates, rather than calling `add_task` 7 times in a loop. | **PASSED** |
| **5.2** | `"Schedule French class at 6 PM on November 12, 14, and 16."` | Calls `add_tasks` in a single call with all 3 occurrences. | **PASSED** |
| **5.3** | `"Schedule morning meditation at 6:00 AM every day."` | Asks: *"Until when would you like to repeat this, or how many occurrences should I create?"* before creating. | **PASSED** |

### 6. Overlap Detection & Conflict Handling
*Goal: Tests database collision warnings.*

| # | Prompt to Test | Expected Behavior | Status |
| :--- | :--- | :--- | :---: |
| **6.1** | `"Schedule a doctor appointment tomorrow from 3:30 PM to 4:30 PM."` *(conflicts with 3 PM task)* | Backend returns `OVERLAP_WARNING`; AI informs user of conflict with "project review" and asks how to proceed. | **PASSED** |

### 7. Search, Inspection, & Counting
*Goal: Tests tool selection preference (`find_tasks`, `count_tasks` instead of dumping all with `get_tasks`).*

| # | Prompt to Test | Expected Behavior | Status |
| :--- | :--- | :--- | :---: |
| **7.1** | `"Do I have any tasks or meetings scheduled for tomorrow?"` | Calls `find_tasks` with tomorrow's date filter. | **PASSED** |
| **7.2** | `"How many incomplete tasks do I have scheduled for this week?"` | Calls `count_tasks` with date range and `completed: false`. | **PASSED** |
| **7.3** | `"Find my gym workouts."` | Calls `find_tasks` with `title_contains: "gym"`. | **PASSED** |

### 8. Update & Reschedule Policy (No Duplicate Creations)
*Goal: Ensures rescheduling uses `update_task` preserving ID rather than delete + add.*

| # | Prompt to Test | Expected Behavior | Status |
| :--- | :--- | :--- | :---: |
| **8.1** | `"Reschedule my project review tomorrow to 5:00 PM."` | First calls `find_tasks` to identify task ID, then calls `update_task` with new time. | **PASSED** |
| **8.2** | `"Mark my project review as completed."` | Calls `find_tasks` ➔ calls `update_task` with `is_completed: true`. | **PASSED** |

### 9. Deletion Safety
*Goal: Tests single delete vs batch delete.*

| # | Prompt to Test | Expected Behavior | Status |
| :--- | :--- | :--- | :---: |
| **9.1** | `"Delete my dentist appointment."` | Identifies single task and calls `delete_task`. | **PASSED** |
| **9.2** | `"Delete all my gym workouts."` | Calls `delete_tasks` (or confirms batch removal). | **PASSED** |

### 10. Multi-Tool Compound Command
*Goal: Tests multi-step reasoning.*

| # | Prompt to Test | Expected Behavior | Status |
| :--- | :--- | :--- | :---: |
| **10.1** | `"Mark my 3 PM review as done, and schedule a follow-up meeting with team for the day after tomorrow at 11 AM."` | Executes both `update_task` and `add_task`, then confirms both actions in final response. | **PASSED** |

---

## Performance & Load Benchmarks

CalenAI's Go backend was tested under rigorous concurrency and throughput benchmarks using Go's testing framework (`testing.B` and concurrent goroutine workers).

### Concurrency & Throughput Stress Test (1,000 Requests per Concurrency Tier)

| Concurrency Tier | Total Requests | Duration | Throughput (RPS) | Latency (Mean) | Latency (p50) | Latency (p95) | Latency (p99) | Error Rate |
| :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **1 Worker** | 1,000 | 147.5 ms | **6,780.97 req/s** | 0.147 ms | 0.000 ms | 1.012 ms | 1.360 ms | **0.00%** |
| **10 Workers** | 1,000 | 117.5 ms | **8,509.94 req/s** | 1.154 ms | 1.053 ms | 2.766 ms | 3.710 ms | **0.00%** |
| **25 Workers** | 1,000 | 115.4 ms | **8,665.89 req/s** | 2.804 ms | 2.683 ms | 6.458 ms | 8.058 ms | **0.00%** |
| **50 Workers** | 1,000 | 125.0 ms | **7,997.62 req/s** | 5.593 ms | 4.686 ms | 18.424 ms | 32.577 ms | **0.00%** |

> [!NOTE]
> Tests executed against the full HTTP router pipeline alternating between task queries (`GET /api/tasks`), endpoint discovery (`GET /api`), and session creation (`POST /api/sessions`). Peak throughput reached **8,665 requests/sec** with zero dropped connections.

### Micro-Benchmark Analysis (per Operation)

| Benchmark Target | Operations Tested | Time / Operation | Memory / Op | Allocations / Op |
| :--- | :---: | :---: | :---: | :---: |
| `GET /api` (Health & Discovery) | 591,499 ops | **2,128 ns** *(2.1 µs)* | 1,804 B/op | 21 allocs/op |
| `PUT /api/tasks/:id` (Update Task) | 38,464 ops | **29,336 ns** *(29.3 µs)* | 9,120 B/op | 144 allocs/op |
| `POST /api/sessions` (Create Session) | 44,275 ops | **31,106 ns** *(31.1 µs)* | 6,466 B/op | 77 allocs/op |
| `POST /api/tasks` (Validation + Insert) | 7,407 ops | **862,929 ns** *(0.86 ms)* | 13,879 B/op | 176 allocs/op |
| `GET /api/tasks` (50 Seeded Tasks + JSON) | 1,046 ops | **1,596,878 ns** *(1.59 ms)* | 595,744 B/op | 10,988 allocs/op |
| `applyTaskFilters` (Query Engine on 100 rows) | 142 ops | **8,272,141 ns** *(8.27 ms)* | 1,323,089 B/op | 131,628 allocs/op |

### Running the Performance Benchmarks Locally

To execute the performance suite and benchmark metrics yourself:

```bash
cd backend
go test -v -bench=Benchmark -benchmem .
```

### High-Volume Concurrent Load Test (20,000 Production-Mix Requests)

To simulate intense, realistic production traffic, a sustained stress test of **20,000 requests** across 4 concurrent tiers (25, 50, 100, and 200 virtual users) was executed with an active production traffic mix:
- **50%** Calendar & Task Retrieval (`GET /api/tasks`)
- **20%** AI Engine Context Queries (`GET /api/context`)
- **15%** Health & Routing Pings (`GET /api`)
- **10%** Dynamic Event Scheduling & Validation (`POST /api/tasks`)
- **5%** Chat Session Provisioning (`POST /api/sessions`)

| Concurrency Tier | Total Requests | Elapsed Time | Throughput (RPS) | Mean Latency | p50 Latency | p90 Latency | p95 Latency | p99 Latency | Server Errors |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **Medium Sustained (25 VU)** | 5,000 | 3.75 s | **1,333.13 req/s** | 17.76 ms | 11.75 ms | 42.45 ms | 53.62 ms | 92.08 ms | **0 (0.00%)** |
| **High Concurrent (50 VU)** | 5,000 | 9.42 s | **530.88 req/s** | 84.37 ms | 55.52 ms | 163.44 ms | 281.20 ms | 785.82 ms | **0 (0.00%)** |
| **Peak Burst (100 VU)** | 5,000 | 15.04 s | **332.36 req/s** | 247.03 ms | 117.32 ms | 398.51 ms | 1,206.20 ms | 3,251.64 ms | **0 (0.00%)** |
| **Extreme Stress (200 VU)** | 5,000 | 20.82 s | **240.13 req/s** | 690.80 ms | 363.62 ms | 1,536.61 ms | 2,647.38 ms | 6,358.46 ms | **0 (0.00%)** |

> [!TIP]
> Under extreme concurrency (up to 200 concurrent clients), the backend sustained a **0.00% server error rate** across all 20,000 requests. Intentional task collision tests verified that duplicate slot bookings reliably returned `409 Conflict` rather than corrupted data or database deadlocks.

To run the high-volume load test suite:

```bash
cd backend
go test -v -run=TestHeavyLoadSimulation -timeout=5m .
```

---

## API Reference

The backend exposes a REST API on `/api`:

### System & Health Checks
- `GET /` — API health check and active service status.
- `GET /api` — Returns available API endpoints.
- `GET /favicon.ico` — Returns `204 No Content`.

### Tasks

#### 1. Retrieve Tasks
```http
GET /api/tasks
```
**Response (`200 OK`):**
```json
[
  {
    "id": 1,
    "title": "Design Review",
    "desc": "Walkthrough calendar UI updates",
    "is_completed": false,
    "start_date_time": "2026-11-06T15:00:00Z",
    "end_date_time": "2026-11-06T16:00:00Z",
    "created_at": "2026-11-05T12:00:00Z",
    "updated_at": "2026-11-05T12:00:00Z"
  }
]
```

#### 2. Create Task
```http
POST /api/tasks
Content-Type: application/json
X-Timezone: Asia/Kolkata

{
  "title": "Gym Workout",
  "desc": "Cardio and upper body",
  "start_date_time": "2026-11-06T07:00:00",
  "end_date_time": "2026-11-06T08:00:00"
}
```
**Response (`201 Created`):**
```json
{
  "id": 2,
  "title": "Gym Workout",
  "desc": "Cardio and upper body",
  "is_completed": false,
  "start_date_time": "2026-11-06T07:00:00+05:30",
  "end_date_time": "2026-11-06T08:00:00+05:30"
}
```

#### 3. Update Task
```http
PUT /api/tasks/:id
Content-Type: application/json

{
  "title": "Gym Workout - Leg Day",
  "is_completed": true
}
```

#### 4. Delete Task
```http
DELETE /api/tasks/:id
```
**Response (`200 OK`):** `{"message": "Task deleted"}`

---

### Chat & Sessions

#### 5. List Sessions
```http
GET /api/sessions
```

#### 6. Create Session
```http
POST /api/sessions
```

#### 7. Get Session Messages
```http
GET /api/sessions/:id
```

#### 8. Send Chat Message (Triggers AI Engine)
```http
POST /api/chat/:session_id
Content-Type: application/json
X-Timezone: Asia/Kolkata
X-Client-Time: 2026-11-05T13:00:00.000Z

{
  "message": "Reschedule my leg day workout to 8 AM tomorrow."
}
```
**Response (`200 OK`):**
```json
{
  "response": "I've rescheduled your Leg Day workout to tomorrow at 8:00 AM."
}
```

---

## Deployment Guide

### Deploying the Backend on Render

1. Create a **New Web Service** connected to your repository on [Render](https://render.com).
2. Set the configuration:
   - **Root Directory**: `backend`
   - **Environment**: `Go`
   - **Build Command**: `go build -o server .`
   - **Start Command**: `./server`
3. In the **Environment Variables** tab, define:
   - `OPENROUTER_API_KEY`: `your_key`
   - `OPENROUTER_MODEL`: `nvidia/nemotron-3-ultra-550b-a55b:free`
   - `GIN_MODE`: `release`
   - `DATABASE_URL`: Your Supabase pooler connection URI (`postgresql://postgres.[REF]:[PASS]@[HOST]:6543/postgres?sslmode=require`).
   - `PORT`: `10000` (Render binds to this automatically).

### Deploying the Frontend on Vercel

1. Import the repository into [Vercel](https://vercel.com).
2. Configure project settings:
   - **Framework Preset**: `Vite`
   - **Root Directory**: `frontend`
   - **Build Command**: `npm run build`
   - **Output Directory**: `dist`
3. In the **Environment Variables** section, add:
   - `VITE_API_URL`: `https://your-render-backend.onrender.com/api`
4. Deploy. Once deployed, ensure your production Vercel URL is permitted in the CORS `AllowOrigins` list in `backend/main.go`.

---

## Troubleshooting & FAQ

### 1. `lookup db.[REF].supabase.co: no such host`
- **Cause**: Supabase direct database hostnames resolve via IPv6 only. If your local network or hosting provider lacks IPv6 support, Go DNS lookups will fail.
- **Solution**: Use the **Supabase Connection Pooler** host (`aws-0-[region].pooler.supabase.com` on port `6543`) which is IPv4-compatible.

### 2. `prepared statement "stmtcache_..." already exists (SQLSTATE 42P05)`
- **Cause**: Transaction connection poolers (PgBouncer) share connections across requests, conflicting with GORM's default prepared statement caching.
- **Solution**: Handled automatically in `backend/main.go` via `PreferSimpleProtocol: true`.

### 3. `net/url: invalid userinfo`
- **Cause**: Database password contains unencoded special characters (`@`, `#`, `$`, `%`, etc.).
- **Solution**: Either URL-encode the special characters in the URI or use the PostgreSQL key-value DSN format (`host=... user=... password=...`).

### 4. Overlap Warning during Task Creation
- **Cause**: The requested time slot conflicts with an existing, active task.
- **Behavior**: The AI intercepts the conflict and responds asking if you would like to keep both, reschedule, or cancel. If you confirm "keep both", it repeats the call with `force: true`.

---

## Roadmap & Engineering Reliability Progress

Derived from [backend/ToDo.md](backend/ToDo.md):

### Completed Items
- [x] **Tool Recursion / Infinite Loop Protection**: Ceiling limit of `maxToolCalls = 10` with depth tracking inside `callOpenRouterWithTools` to prevent runaway recursive tool loops and timeouts.
- [x] **Multi-Tool Message Handling Cleanup**: Cleaned up turn management: the assistant tool call message is appended once to conversation history followed by all individual tool execution results (RFC/OpenAI/OpenRouter compliant).
- [x] **Bulk Task Filtering System**: Implemented flexible `applyTaskFilters` query builder supporting date ranges (`start_date`, `end_date`), title substrings (`title_contains`), and completion states (`completed`). Powers `find_tasks`, `count_tasks`, and `delete_tasks`.
- [x] **Advanced Task Search**: Natural language task lookups via `find_tasks`. System prompt policy enforces inspecting matching tasks via `find_tasks` or `count_tasks` rather than dumping all tasks with `get_tasks`.
- [x] **Batch Creation Optimization (`add_tasks`)**: Multi-day and recurring events (e.g., daily standup for 30 days) are scheduled in a single request instead of 30 recursive tool cycles. Includes conflict detection and `force: true` override support.
- [x] **Database Hardening & Security**:
  - Partial unique index on active records (`idx_task_unique_active WHERE deleted_at IS NULL`), preventing soft-deleted tasks from colliding with new ones.
  - Strict unsigned integer ID validation (`strconv.ParseUint`) on all path parameters to block SQL injection attempts.
  - Supabase PgBouncer pooler compatibility using `PreferSimpleProtocol: true` to prevent prepared statement cache collisions (`42P05`).
- [x] **Authoritative Client Time Awareness**: Browser sends dynamic `X-Client-Time` and `X-Timezone` headers on every API interaction, eliminating timezone discrepancies between client and cloud.

### In Progress & Future Enhancements
- [ ] **Safe Deletion Confirmation Workflow**: Currently prompt-enforced (AI verifies intent before calling `delete_tasks`). Future: Dedicated interactive two-step token confirmation state in chat session storage.
- [ ] **User Ownership & Multi-Tenancy**: Add `user_id` foreign keys to `Task` and `Session` models, combined with JWT/session authentication middleware for multi-user isolation.
- [ ] **Soft-Delete Undo Recovery**: Add `restore_task` / `undo_delete` tool and endpoint to un-set `deleted_at` timestamps for effortless recovery.

---

## Contributing

1. Fork the repository.
2. Create your feature branch: `git checkout -b feature/my-feature`.
3. Commit your changes: `git commit -m "Add my feature"`.
4. Push to the branch: `git push origin feature/my-feature`.
5. Open a Pull Request.

---

## License

This project is licensed under the [MIT License](LICENSE).
