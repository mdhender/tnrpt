// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package stages_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mdhender/tnrpt/model"
	store "github.com/mdhender/tnrpt/stores/sqlite"
)

func TestWorkerService_ClaimJob_AtomicLocking(t *testing.T) {
	ctx := context.Background()
	sqlStore, err := store.NewSQLiteStore()
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	defer sqlStore.Close()

	batchID, err := sqlStore.InsertUploadBatch(ctx, &model.UploadBatch{
		Game:      "0301",
		ClanNo:    "0512",
		TurnNo:    89912,
		CreatedBy: "test",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("insert batch: %v", err)
	}

	rfID, err := sqlStore.InsertReportFileWithBatch(ctx, &model.ReportFile{
		Game:      "0301",
		ClanNo:    "0512",
		TurnNo:    89912,
		Name:      "test.docx",
		SHA256:    "abc123",
		Mime:      "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		CreatedAt: time.Now().UTC(),
		FsPath:    "batches/1/test.docx",
		BatchID:   &batchID,
	})
	if err != nil {
		t.Fatalf("insert report file: %v", err)
	}

	_, err = sqlStore.InsertWork(ctx, &model.Work{
		ReportFileID: rfID,
		Stage:        model.WorkStageExtract,
		Status:       model.WorkStatusQueued,
		Attempt:      0,
		AvailableAt:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("insert work: %v", err)
	}

	const numWorkers = 10
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	claimedCount := 0
	var mu sync.Mutex

	for i := 0; i < numWorkers; i++ {
		workerID := i
		go func() {
			defer wg.Done()
			work, err := sqlStore.ClaimWork(ctx, model.WorkStageExtract, "worker-"+string(rune('A'+workerID)))
			if err != nil {
				t.Errorf("worker %d: claim error: %v", workerID, err)
				return
			}
			if work != nil {
				mu.Lock()
				claimedCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if claimedCount != 1 {
		t.Errorf("expected exactly 1 worker to claim the job, got %d", claimedCount)
	}
}

func TestWorkerService_ClaimJob_ReturnsNilWhenNoWork(t *testing.T) {
	ctx := context.Background()
	sqlStore, err := store.NewSQLiteStore()
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	defer sqlStore.Close()

	work, err := sqlStore.ClaimWork(ctx, model.WorkStageExtract, "test-worker")
	if err != nil {
		t.Fatalf("claim work: %v", err)
	}
	if work != nil {
		t.Errorf("expected nil work when no jobs available, got %+v", work)
	}
}

func TestResetFailedWork_ResetsFailedJobs(t *testing.T) {
	ctx := context.Background()
	sqlStore, err := store.NewSQLiteStore()
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	defer sqlStore.Close()

	batchID, err := sqlStore.InsertUploadBatch(ctx, &model.UploadBatch{
		Game:      "0301",
		ClanNo:    "0512",
		TurnNo:    89912,
		CreatedBy: "test",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("insert batch: %v", err)
	}

	rfID1, err := sqlStore.InsertReportFileWithBatch(ctx, &model.ReportFile{
		Game:      "0301",
		ClanNo:    "0512",
		TurnNo:    89912,
		Name:      "file1.docx",
		SHA256:    "hash1",
		Mime:      "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		CreatedAt: time.Now().UTC(),
		FsPath:    "batches/1/file1.docx",
		BatchID:   &batchID,
	})
	if err != nil {
		t.Fatalf("insert report file 1: %v", err)
	}

	rfID2, err := sqlStore.InsertReportFileWithBatch(ctx, &model.ReportFile{
		Game:      "0301",
		ClanNo:    "0512",
		TurnNo:    89912,
		Name:      "file2.docx",
		SHA256:    "hash2",
		Mime:      "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		CreatedAt: time.Now().UTC(),
		FsPath:    "batches/1/file2.docx",
		BatchID:   &batchID,
	})
	if err != nil {
		t.Fatalf("insert report file 2: %v", err)
	}

	rfID3, err := sqlStore.InsertReportFileWithBatch(ctx, &model.ReportFile{
		Game:      "0301",
		ClanNo:    "0512",
		TurnNo:    89912,
		Name:      "file3.docx",
		SHA256:    "hash3",
		Mime:      "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		CreatedAt: time.Now().UTC(),
		FsPath:    "batches/1/file3.docx",
		BatchID:   &batchID,
	})
	if err != nil {
		t.Fatalf("insert report file 3: %v", err)
	}

	work1ID, err := sqlStore.InsertWork(ctx, &model.Work{
		ReportFileID: rfID1,
		Stage:        model.WorkStageExtract,
		Status:       model.WorkStatusQueued,
		Attempt:      0,
		AvailableAt:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("insert work 1: %v", err)
	}

	work2ID, err := sqlStore.InsertWork(ctx, &model.Work{
		ReportFileID: rfID2,
		Stage:        model.WorkStageExtract,
		Status:       model.WorkStatusQueued,
		Attempt:      0,
		AvailableAt:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("insert work 2: %v", err)
	}

	work3ID, err := sqlStore.InsertWork(ctx, &model.Work{
		ReportFileID: rfID3,
		Stage:        model.WorkStageParse,
		Status:       model.WorkStatusQueued,
		Attempt:      0,
		AvailableAt:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("insert work 3: %v", err)
	}

	err = sqlStore.FinishWork(ctx, work1ID, model.WorkStatusFailed, "PARSE_ERROR", "syntax error")
	if err != nil {
		t.Fatalf("finish work 1: %v", err)
	}
	err = sqlStore.FinishWork(ctx, work2ID, model.WorkStatusFailed, "DOCX_CORRUPT", "invalid format")
	if err != nil {
		t.Fatalf("finish work 2: %v", err)
	}
	err = sqlStore.FinishWork(ctx, work3ID, model.WorkStatusFailed, "PARSE_ERROR", "syntax error")
	if err != nil {
		t.Fatalf("finish work 3: %v", err)
	}

	failedBefore, err := sqlStore.GetFailedWork(ctx, model.WorkStageExtract)
	if err != nil {
		t.Fatalf("get failed work before: %v", err)
	}
	if len(failedBefore) != 2 {
		t.Errorf("expected 2 failed extract jobs before reset, got %d", len(failedBefore))
	}

	resetCount, err := sqlStore.ResetFailedWork(ctx, model.WorkStageExtract)
	if err != nil {
		t.Fatalf("reset failed work: %v", err)
	}
	if resetCount != 2 {
		t.Errorf("expected 2 jobs reset, got %d", resetCount)
	}

	failedAfter, err := sqlStore.GetFailedWork(ctx, model.WorkStageExtract)
	if err != nil {
		t.Fatalf("get failed work after: %v", err)
	}
	if len(failedAfter) != 0 {
		t.Errorf("expected 0 failed extract jobs after reset, got %d", len(failedAfter))
	}

	claimedJobs := 0
	for i := 0; i < 3; i++ {
		work, err := sqlStore.ClaimWork(ctx, model.WorkStageExtract, "test-worker")
		if err != nil {
			t.Fatalf("claim work %d: %v", i, err)
		}
		if work != nil {
			claimedJobs++
		}
	}
	if claimedJobs != 2 {
		t.Errorf("expected 2 jobs to be claimable after reset, got %d", claimedJobs)
	}

	parseFailedAfter, err := sqlStore.GetFailedWork(ctx, model.WorkStageParse)
	if err != nil {
		t.Fatalf("get failed parse work: %v", err)
	}
	if len(parseFailedAfter) != 1 {
		t.Errorf("expected 1 failed parse job (unaffected by reset), got %d", len(parseFailedAfter))
	}
}

func TestResetFailedWork_ClearsErrorFields(t *testing.T) {
	ctx := context.Background()
	sqlStore, err := store.NewSQLiteStore()
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	defer sqlStore.Close()

	batchID, err := sqlStore.InsertUploadBatch(ctx, &model.UploadBatch{
		Game:      "0301",
		ClanNo:    "0512",
		TurnNo:    89912,
		CreatedBy: "test",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("insert batch: %v", err)
	}

	rfID, err := sqlStore.InsertReportFileWithBatch(ctx, &model.ReportFile{
		Game:      "0301",
		ClanNo:    "0512",
		TurnNo:    89912,
		Name:      "test.docx",
		SHA256:    "hash123",
		Mime:      "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		CreatedAt: time.Now().UTC(),
		FsPath:    "batches/1/test.docx",
		BatchID:   &batchID,
	})
	if err != nil {
		t.Fatalf("insert report file: %v", err)
	}

	workID, err := sqlStore.InsertWork(ctx, &model.Work{
		ReportFileID: rfID,
		Stage:        model.WorkStageExtract,
		Status:       model.WorkStatusQueued,
		Attempt:      0,
		AvailableAt:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("insert work: %v", err)
	}

	claimed, err := sqlStore.ClaimWork(ctx, model.WorkStageExtract, "worker-1")
	if err != nil {
		t.Fatalf("claim work: %v", err)
	}
	if claimed == nil {
		t.Fatal("expected to claim work")
	}

	err = sqlStore.FinishWork(ctx, workID, model.WorkStatusFailed, "TEST_ERROR", "test error message")
	if err != nil {
		t.Fatalf("finish work: %v", err)
	}

	failedBefore, err := sqlStore.GetFailedWork(ctx, model.WorkStageExtract)
	if err != nil {
		t.Fatalf("get failed work: %v", err)
	}
	if len(failedBefore) != 1 {
		t.Fatalf("expected 1 failed job, got %d", len(failedBefore))
	}
	if failedBefore[0].ErrorCode == nil || *failedBefore[0].ErrorCode != "TEST_ERROR" {
		t.Errorf("expected error code 'TEST_ERROR', got %v", failedBefore[0].ErrorCode)
	}

	_, err = sqlStore.ResetFailedWork(ctx, model.WorkStageExtract)
	if err != nil {
		t.Fatalf("reset failed work: %v", err)
	}

	reclaimedWork, err := sqlStore.ClaimWork(ctx, model.WorkStageExtract, "worker-2")
	if err != nil {
		t.Fatalf("reclaim work: %v", err)
	}
	if reclaimedWork == nil {
		t.Fatal("expected to reclaim work after reset")
	}
	if reclaimedWork.ErrorCode != nil {
		t.Errorf("expected error_code to be cleared, got %v", *reclaimedWork.ErrorCode)
	}
	if reclaimedWork.ErrorMessage != nil {
		t.Errorf("expected error_message to be cleared, got %v", *reclaimedWork.ErrorMessage)
	}
	if reclaimedWork.LockedBy == nil || *reclaimedWork.LockedBy != "worker-2" {
		t.Errorf("expected locked_by to be 'worker-2', got %v", reclaimedWork.LockedBy)
	}
	if reclaimedWork.Status != model.WorkStatusRunning {
		t.Errorf("expected status 'running', got %q", reclaimedWork.Status)
	}
}
