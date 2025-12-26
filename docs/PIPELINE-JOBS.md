# Background Job Pipeline (CLI Implementation)

This document defines the **background job pipeline** for report processing using SQLite-backed staged work queue.

**Status**: Design finalized. Implementation pending.

---

## MVP Scope

The next TribeNet turn runs in a few days. This scope gets us a working system:

**In Scope (MVP)**:
- CLI ingest: file → DB + filesystem
- CLI extract worker: DOCX → text file (for scrubber debugging)
- CLI parse worker: text → model tables
- Render/download form (players need maps)
- Schema additions (clean CREATE, no ALTER — DB rebuilt each deployment)

**Deferred**:
- GM upload form (GM can email reports for now)
- Automatic retries with backoff
- work_log table (use stdout logging initially)
- Web status polling / toast notifications

**Deployment Model**: Container rebuilds DB from scratch; existing report files batch-loaded during deployment.

**Schema Note**: Currently have both `model/schema.sql` and `web/store/schema.sql`. These need consolidation into a single schema file. Track this as tech debt.

---

## Architecture

### High-Level Flow

```
File → CLI ingest → Report File + Work Queue → Worker Claims Job → Execute Stage
                    ↓
                    report_files (with fs_path, batch_id)
                    upload_batches
                    work
```

### Implementation Structure

```
cmd/tnrpt/
├── main.go
└── commands/
    └── pipeline.go          (cobra command: ingest, work, status)

pipelines/stages/
├── ingest.go                (IngestService: file → DB)
├── worker.go                (WorkerService: claim → execute → finish)
├── types.go                 (UploadBatch, Work)
└── errors.go                (error types)

model/
├── schema.sql               (includes: upload_batches, work, report_files columns)
├── store.go                 (batch, work repositories)
└── types.go                 (UploadBatch, Work types)
```

---

## Schema

Schema rebuilt each deployment — use clean CREATE statements, no ALTER.

### upload_batches

Groups multiple files in one ingest operation.

```sql
CREATE TABLE IF NOT EXISTS upload_batches
(
  id         INTEGER PRIMARY KEY,
  game       TEXT    NOT NULL,
  clan_no    TEXT    NOT NULL,
  turn_no    INTEGER NOT NULL,
  created_by TEXT,                -- CLI user or web session
  created_at TEXT    NOT NULL     -- ISO8601 UTC
);

CREATE INDEX IF NOT EXISTS idx_upload_batches_game_turn
  ON upload_batches(game, turn_no, clan_no);
```

### report_files (columns)

Add these columns to the existing report_files table definition:

```sql
-- In report_files CREATE TABLE:
fs_path   TEXT,     -- Relative to data-dir; e.g., "batches/1/0512.docx"
batch_id  INTEGER REFERENCES upload_batches(id) ON DELETE CASCADE
```

**Invariant**: If `fs_path` is non-NULL, it points to an existing, readable file.

### work (job queue)

Single queue for all pipeline stages.

```sql
CREATE TABLE IF NOT EXISTS work
(
  id             INTEGER PRIMARY KEY,
  report_file_id INTEGER NOT NULL REFERENCES report_files (id) ON DELETE CASCADE,

  stage          TEXT    NOT NULL,                  -- 'extract', 'parse'
  status         TEXT    NOT NULL DEFAULT 'queued', -- queued|running|ok|failed

  attempt        INTEGER NOT NULL DEFAULT 0,
  available_at   TEXT    NOT NULL,                  -- ISO8601 UTC
  locked_by      TEXT,                              -- worker ID
  locked_at      TEXT,                              -- ISO8601 UTC
  started_at     TEXT,                              -- first execution time
  finished_at    TEXT,                              -- ISO8601 UTC

  error_code     TEXT,                              -- e.g., "PARSE_SYNTAX_ERROR"
  error_message  TEXT,

  UNIQUE (report_file_id, stage)
);

CREATE INDEX IF NOT EXISTS idx_work_ready 
  ON work(status, stage, available_at);

CREATE INDEX IF NOT EXISTS idx_work_file
  ON work(report_file_id);
```

**Lifecycle**:
1. `ingest` creates row: stage=`extract`, status=`queued`, attempt=0, available_at=now
2. Stage worker claims: status→`running`, locked_by=worker_id, attempt++
3. Success: status→`ok`, finished_at=now; create next stage job if applicable
4. Failure: status→`failed`, finished_at=now, error_code/message set

---

## Domain Types

### model.UploadBatch

```go
type UploadBatch struct {
  ID        int64
  Game      string    // e.g., "0301"
  ClanNo    string    // e.g., "0512"
  TurnNo    int       // e.g., 89912 (year 899, month 12)
  CreatedBy string    // CLI user or web session
  CreatedAt time.Time
}
```

### model.Work

```go
type Work struct {
  ID            int64
  ReportFileID  int64
  Stage         string    // "extract", "parse"
  Status        string    // "queued", "running", "ok", "failed"
  Attempt       int
  AvailableAt   time.Time
  LockedBy      *string   // worker ID
  LockedAt      *time.Time
  StartedAt     *time.Time
  FinishedAt    *time.Time
  ErrorCode     *string
  ErrorMessage  *string
}
```

---

## Services

### IngestService (pipelines/stages/ingest.go)

Writes file durably and queues extraction.

```go
type IngestService struct {
  store   *model.Store
  dataDir string  // --data-dir flag
  fs      afero.Fs // injected for testing
}

// IngestFile does:
// 1. Read file and compute SHA-256
// 2. Check for duplicates in report_files
// 3. Ensure batch exists
// 4. Write file to final location
// 5. Insert report_files row with fs_path
// 6. Insert work row for 'extract'
func (s *IngestService) IngestFile(ctx context.Context, 
  batchID int64, game, clan string, turnNo int, 
  filename string, data []byte) error
```

**Behavior**:
- Duplicate file (same SHA-256): **idempotent no-op** — log and continue, not an error
- DOCX file: queue for extract stage
- TEXT file (`.txt`): queue directly for parse stage (skip extract)

**Errors**:
- `ErrWriteFile`: I/O error
- `ErrDatabase`: DB error

### WorkerService (pipelines/stages/worker.go)

Claims jobs and executes stages.

```go
type WorkerService struct {
  store    *model.Store
  dataDir  string
  workerID string      // hostname:pid or equivalent
  fs       afero.Fs    // injected for testing
}

// ClaimJob atomically claims exactly one job.
// Returns nil if no work available.
func (w *WorkerService) ClaimJob(ctx context.Context, 
  stage string) (*model.Work, error)

// ExecuteExtract reads DOCX from fs_path, 
// extracts text, writes to intermediate text file.
// Text file saved for scrubber debugging.
// If fs_path is already a .txt file, skip extraction and queue parse directly.
func (w *WorkerService) ExecuteExtract(ctx context.Context, 
  job *model.Work, rf *model.ReportFile) error

// ExecuteParse reads extracted text, 
// runs bistre parser, stores to model tables.
func (w *WorkerService) ExecuteParse(ctx context.Context, 
  job *model.Work, rf *model.ReportFile) error

// FinishJob marks job ok or failed.
func (w *WorkerService) FinishJob(ctx context.Context, 
  job *model.Work, result WorkResult) error
```

**Key Properties**:
- ClaimJob uses atomic UPDATE...WHERE...RETURNING
- Execute stages run **outside** transaction
- Extract saves text file for scrubber review
- On success, extract creates 'parse' work row

---

## CLI Commands

### tnrpt pipeline ingest

Ingest files into a batch and queue for processing.

```bash
# Single file
tnrpt pipeline ingest \
  --db ./data/tnrpt.db \
  --data-dir ./data \
  --game 0301 \
  --clan 0512 \
  --turn 89912 \
  path/to/0512.docx

# Batch of files
tnrpt pipeline ingest \
  --db ./data/tnrpt.db \
  --data-dir ./data \
  --game 0301 \
  --clan 0512 \
  --turn 89912 \
  path/to/*.docx

# Output:
# Batch ID: 42
# Ingested 3 files: 0512.docx, 0513.docx, 0514.docx
# Queued for extraction
```

**File Naming Convention**:
```
GGGG.YYYY-MM.CCCC.docx        # turn report (DOCX)
GGGG.YYYY-MM.CCCC.report.txt  # report extract (TEXT)
```
Where: GGGG=game, YYYY-MM=turn (year-month), CCCC=clan

Files written to:
```
data-dir/batches/{batch_id}/GGGG.YYYY-MM.CCCC.docx
data-dir/batches/{batch_id}/GGGG.YYYY-MM.CCCC.report.txt
```

### tnrpt pipeline work

Worker loop: claim jobs, execute, finish.

```bash
# Extract worker
tnrpt pipeline work extract \
  --db ./data/tnrpt.db \
  --data-dir ./data \
  --poll-interval 5s

# Parse worker
tnrpt pipeline work parse \
  --db ./data/tnrpt.db \
  --data-dir ./data \
  --poll-interval 5s

# All stages (sequential)
tnrpt pipeline work all \
  --db ./data/tnrpt.db \
  --data-dir ./data \
  --poll-interval 5s

# Retry failed jobs for a stage (ignores backoff timing)
tnrpt pipeline work extract \
  --db ./data/tnrpt.db \
  --data-dir ./data \
  --retry-failed
```

**Flags**:
- `--poll-interval`: Time between claim attempts when idle (default: 5s)
- `--retry-failed`: Reset all failed jobs for this stage to queued before starting (default: false)

**Implementation**:
```go
for {
  job, err := workerService.ClaimJob(ctx, stage)
  if err != nil {
    log.Printf("claim failed: %v", err)
    time.Sleep(pollInterval)
    continue
  }
  if job == nil {
    time.Sleep(pollInterval)
    continue
  }

  rf, err := store.GetReportFile(ctx, job.ReportFileID)
  if err != nil {
    log.Printf("get report file failed: %v", err)
    workerService.FinishJob(ctx, job, WorkResult{Error: err})
    continue
  }

  var execErr error
  switch job.Stage {
  case "extract":
    execErr = workerService.ExecuteExtract(ctx, job, rf)
  case "parse":
    execErr = workerService.ExecuteParse(ctx, job, rf)
  }

  if execErr != nil {
    log.Printf("stage %s failed: %v", job.Stage, execErr)
    workerService.FinishJob(ctx, job, WorkResult{Error: execErr})
  } else {
    log.Printf("stage %s ok for file %d", job.Stage, job.ReportFileID)
    workerService.FinishJob(ctx, job, WorkResult{})
  }
}
```

### tnrpt pipeline status

Query batch and work status, list failed jobs.

```bash
# Summary for a batch
tnrpt pipeline status \
  --db ./data/tnrpt.db \
  --batch-id 42

# Output:
# Batch 42 (game=0301, clan=0512, turn=89912)
# Created: 2025-01-15T10:30:00Z
#
# Work Summary:
#   extract: 3 ok, 0 running, 0 queued, 0 failed
#   parse:   3 ok, 0 running, 0 queued, 0 failed

# List all failed jobs
tnrpt pipeline status \
  --db ./data/tnrpt.db \
  --failed

# Output:
# Failed Jobs:
#   ID=7  stage=extract  file=0301.899-12.0512.docx  error=DOCX_CORRUPT
#   ID=12 stage=parse    file=0301.899-12.0513.docx  error=PARSE_SYNTAX_ERROR
#
# To retry: tnrpt pipeline work extract --retry-failed

# List failed jobs for a specific stage
tnrpt pipeline status \
  --db ./data/tnrpt.db \
  --failed \
  --stage extract
```

**Flags**:
- `--batch-id`: Show summary for specific batch
- `--failed`: List failed jobs instead of summary
- `--stage`: Filter failed jobs by stage (extract, parse)

---

## Execution Model

### Job Lifecycle

```
1. INGEST (CLI command)
   ├─ Standardize filename: GGGG.YYYY-MM.CCCC.{docx|report.txt}
   ├─ Compute SHA-256
   ├─ Check for duplicate (same SHA-256) → idempotent no-op if exists
   ├─ Write file to disk
   ├─ Insert report_files (fs_path, batch_id)
   └─ Insert work:
      ├─ DOCX file → stage='extract'
      └─ TEXT file → stage='parse' (skip extraction)

2. EXTRACT (background worker)
   ├─ Claim: UPDATE work SET status='running' (atomic)
   ├─ Check file type:
   │  ├─ DOCX → extract text, write .report.txt file
   │  └─ TEXT → no-op (already extracted)
   ├─ Finish: UPDATE work SET status='ok'
   └─ Create work row for next stage (parse)

3. PARSE (background worker)
   ├─ Claim: UPDATE work SET status='running' (atomic)
   ├─ Execute: text → bistre parser → model tables
   ├─ Finish: UPDATE work SET status='ok'
   └─ Done

4. RENDER (on-demand, foreground — existing web handler)
   ├─ Read parsed tables (unit_extracts, acts, steps, tiles)
   └─ Generate map (no DB writes)
```

### File Layout

```
data-dir/
├─ batches/
│  └─ {batch_id}/
│     ├─ 0301.899-12.0512.docx         # original DOCX (turn report)
│     └─ 0301.899-12.0512.report.txt   # extracted text (for scrubber debugging)
└─ tnrpt.db
```

### Worker Claim (Atomic)

```sql
UPDATE work
SET status     = 'running',
    locked_by  = ?,
    locked_at  = ?,
    started_at = COALESCE(started_at, ?),
    attempt    = attempt + 1
WHERE id = (SELECT id
            FROM work
            WHERE status = 'queued'
              AND stage = ?
              AND available_at <= ?
            ORDER BY id
            LIMIT 1)
RETURNING id, report_file_id, stage, attempt;
```

**Properties**:
- SQLite serializes writes → only one worker claims each job
- If claim returns 0 rows → no work available
- Heavy work (extract, parse) runs **outside** transaction

---

## Error Handling (MVP)

For MVP, no automatic retries. On failure:
1. Mark job status='failed', set error_code/message
2. Log error to stdout
3. Continue to next job

**Retry via CLI**:
```bash
# List failed jobs
tnrpt pipeline status --db ./data/tnrpt.db --failed

# Retry all failed extract jobs
tnrpt pipeline work extract --db ./data/tnrpt.db --data-dir ./data --retry-failed
```

The `--retry-failed` flag runs this SQL before starting the worker loop:
```sql
UPDATE work
SET status     = 'queued',
    locked_by  = NULL,
    locked_at  = NULL,
    error_code = NULL,
    error_message = NULL
WHERE status = 'failed'
  AND stage = ?;
```

---

## Testing Strategy

### In-Memory SQLite + Afero

```go
import "github.com/spf13/afero"

func TestIngestService(t *testing.T) {
  store, err := model.NewStore(context.Background(), ":memory:")
  require.NoError(t, err)
  defer store.Close()

  fs := afero.NewMemMapFs()
  ingest := pipeline.NewIngestService(store, "/data")
  ingest.fs = fs

  require.NoError(t, afero.WriteFile(fs, "/data/test.docx", testDocxBytes, 0644))

  batchID, err := ingest.IngestFile(ctx, 1, "TestGame", "0512", 89912, 
    "test.docx", testDocxBytes)
  require.NoError(t, err)

  work, err := store.GetWorkByFile(ctx, batchID)
  require.Len(t, work, 1)
  require.Equal(t, "extract", work[0].Stage)
  require.Equal(t, "queued", work[0].Status)
}

func TestWorkerClaimJob(t *testing.T) {
  store, _ := model.NewStore(context.Background(), ":memory:")
  defer store.Close()
  fs := afero.NewMemMapFs()

  // ... insert test data ...

  worker := pipeline.NewWorkerService(store, "/data", "test-worker")
  worker.fs = fs

  job, err := worker.ClaimJob(ctx, "extract")
  require.NoError(t, err)
  require.NotNil(t, job)
  require.Equal(t, "running", job.Status)
  require.Equal(t, "test-worker", *job.LockedBy)

  // Verify locked — second claim returns nil
  job2, err := worker.ClaimJob(ctx, "extract")
  require.NoError(t, err)
  require.Nil(t, job2)
}
```

---

## Web Integration

### MVP: Render/Download Only

The existing render/download handler remains unchanged. It reads parsed data from model tables and generates maps.

Players access: `GET /games/{game}/turns/{turn}/clans/{clan}/render`

### Deferred: Upload Form

GM upload form deferred. GM emails reports; operator runs:
```bash
tnrpt pipeline ingest --game 0301 --clan 0512 --turn 89912 *.docx
tnrpt pipeline work all
```

### Future: Full Web Integration

After MVP is stable:
1. `POST /api/upload/batch` → calls IngestService, returns batch_id
2. `GET /api/batches/{id}/status` → queries work summary
3. Template polls for status, shows progress

---

## Implementation Order

1. **Schema**: Add upload_batches, work tables; add fs_path/batch_id to report_files
2. **Model types**: UploadBatch, Work structs and repository methods
3. **IngestService**: File write + DB insert + work queue
4. **WorkerService**: ClaimJob, ExecuteExtract, ExecuteParse, FinishJob
5. **CLI commands**: `pipeline ingest`, `pipeline work`, `pipeline status`
6. **Tests**: In-memory SQLite + Afero
7. **Verify render**: Ensure existing render handler works with parsed data

---

## Summary

This MVP design:

✅ Keeps render/download working (players get maps)  
✅ Eliminates OOM crashes (bounded concurrency)  
✅ Saves extracted text (scrubber debugging)  
✅ CLI-first (easy testing, scripting, batch loading)  
✅ Clean schema (no ALTER, rebuilt each deployment)  
✅ Minimal scope (defer retries, work_log, upload form)  

**This is a job queue, not an event stream.**
