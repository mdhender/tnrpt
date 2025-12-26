// Copyright (c) 2025 Michael D Henderson. All rights reserved.

package stages_test

import (
	"context"
	"testing"
	"time"

	"github.com/mdhender/tnrpt/model"
	"github.com/mdhender/tnrpt/pipelines/stages"
	"github.com/spf13/afero"
)

// mockStore implements stages.IngestStore for testing.
type mockStore struct {
	batches     map[int64]*model.UploadBatch
	reportFiles map[int64]*model.ReportFile
	work        map[int64]*model.Work
	sha256Index map[string]*model.ReportFile

	nextBatchID int64
	nextRFID    int64
	nextWorkID  int64
}

func newMockStore() *mockStore {
	return &mockStore{
		batches:     make(map[int64]*model.UploadBatch),
		reportFiles: make(map[int64]*model.ReportFile),
		work:        make(map[int64]*model.Work),
		sha256Index: make(map[string]*model.ReportFile),
		nextBatchID: 1,
		nextRFID:    1,
		nextWorkID:  1,
	}
}

func (m *mockStore) InsertUploadBatch(_ context.Context, batch *model.UploadBatch) (int64, error) {
	id := m.nextBatchID
	m.nextBatchID++
	batch.ID = id
	m.batches[id] = batch
	return id, nil
}

func (m *mockStore) GetUploadBatch(_ context.Context, id int64) (*model.UploadBatch, error) {
	return m.batches[id], nil
}

func (m *mockStore) GetReportFileBySHA256(_ context.Context, sha256 string) (*model.ReportFile, error) {
	return m.sha256Index[sha256], nil
}

func (m *mockStore) InsertReportFileWithBatch(_ context.Context, rf *model.ReportFile) (int64, error) {
	id := m.nextRFID
	m.nextRFID++
	rf.ID = id
	m.reportFiles[id] = rf
	m.sha256Index[rf.SHA256] = rf
	return id, nil
}

func (m *mockStore) InsertWork(_ context.Context, work *model.Work) (int64, error) {
	id := m.nextWorkID
	m.nextWorkID++
	work.ID = id
	m.work[id] = work
	return id, nil
}

func TestIngestService_IngestFile(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	fs := afero.NewMemMapFs()

	svc := stages.NewIngestService(store, "/data")
	svc.SetFS(fs)

	batchID, err := store.InsertUploadBatch(ctx, &model.UploadBatch{
		Game:      "0301",
		ClanNo:    "0512",
		TurnNo:    89912,
		CreatedBy: "test",
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("insert batch: %v", err)
	}

	docxData := []byte("fake docx content")
	req := stages.IngestRequest{
		Game:     "0301",
		ClanNo:   "0512",
		TurnNo:   89912,
		Filename: "test.docx",
		Data:     docxData,
	}

	result, err := svc.IngestFile(ctx, batchID, req)
	if err != nil {
		t.Fatalf("ingest file: %v", err)
	}
	if result.Duplicate {
		t.Error("expected not duplicate on first ingest")
	}
	if result.ReportFileID == 0 {
		t.Error("expected non-zero report file ID")
	}
	if result.WorkID == 0 {
		t.Error("expected non-zero work ID")
	}

	rf := store.reportFiles[result.ReportFileID]
	if rf == nil {
		t.Fatal("report file not found in store")
	}
	if rf.Name != "0301.899-12.0512.docx" {
		t.Errorf("expected name '0301.899-12.0512.docx', got %q", rf.Name)
	}
	if rf.FsPath != "batches/1/0301.899-12.0512.docx" {
		t.Errorf("expected fs_path 'batches/1/0301.899-12.0512.docx', got %q", rf.FsPath)
	}

	work := store.work[result.WorkID]
	if work == nil {
		t.Fatal("work not found in store")
	}
	if work.Stage != model.WorkStageExtract {
		t.Errorf("expected stage 'extract' for DOCX, got %q", work.Stage)
	}

	exists, err := afero.Exists(fs, "/data/batches/1/0301.899-12.0512.docx")
	if err != nil {
		t.Fatalf("check file exists: %v", err)
	}
	if !exists {
		t.Error("expected file to exist on filesystem")
	}
}

func TestIngestService_DuplicateIsIdempotent(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	fs := afero.NewMemMapFs()

	svc := stages.NewIngestService(store, "/data")
	svc.SetFS(fs)

	batchID, _ := store.InsertUploadBatch(ctx, &model.UploadBatch{
		Game:      "0301",
		ClanNo:    "0512",
		TurnNo:    89912,
		CreatedBy: "test",
		CreatedAt: time.Now().UTC(),
	})

	docxData := []byte("fake docx content")
	req := stages.IngestRequest{
		Game:     "0301",
		ClanNo:   "0512",
		TurnNo:   89912,
		Filename: "test.docx",
		Data:     docxData,
	}

	result1, err := svc.IngestFile(ctx, batchID, req)
	if err != nil {
		t.Fatalf("first ingest: %v", err)
	}

	result2, err := svc.IngestFile(ctx, batchID, req)
	if err != nil {
		t.Fatalf("second ingest: %v", err)
	}
	if !result2.Duplicate {
		t.Error("expected duplicate=true on second ingest")
	}
	if result2.ReportFileID != result1.ReportFileID {
		t.Error("expected same report file ID for duplicate")
	}
	if result2.WorkID != 0 {
		t.Error("expected zero work ID for duplicate (no new work created)")
	}
}

func TestIngestService_TextFileQueuesParseStage(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	fs := afero.NewMemMapFs()

	svc := stages.NewIngestService(store, "/data")
	svc.SetFS(fs)

	batchID, _ := store.InsertUploadBatch(ctx, &model.UploadBatch{
		Game:      "0301",
		ClanNo:    "0512",
		TurnNo:    89912,
		CreatedBy: "test",
		CreatedAt: time.Now().UTC(),
	})

	txtData := []byte("turn report text content")
	req := stages.IngestRequest{
		Game:     "0301",
		ClanNo:   "0512",
		TurnNo:   89912,
		Filename: "test.txt",
		Data:     txtData,
	}

	result, err := svc.IngestFile(ctx, batchID, req)
	if err != nil {
		t.Fatalf("ingest file: %v", err)
	}

	work := store.work[result.WorkID]
	if work == nil {
		t.Fatal("work not found in store")
	}
	if work.Stage != model.WorkStageParse {
		t.Errorf("expected stage 'parse' for TXT, got %q", work.Stage)
	}
}

func TestIngestService_IngestBatch(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	fs := afero.NewMemMapFs()

	svc := stages.NewIngestService(store, "/data")
	svc.SetFS(fs)

	files := []stages.IngestRequest{
		{Filename: "unit1.docx", Data: []byte("docx content 1")},
		{Filename: "unit2.docx", Data: []byte("docx content 2")},
		{Filename: "unit3.txt", Data: []byte("text content 3")},
	}

	batchID, results, err := svc.IngestBatch(ctx, "0301", "0512", 89912, "test-user", files)
	if err != nil {
		t.Fatalf("ingest batch: %v", err)
	}
	if batchID == 0 {
		t.Error("expected non-zero batch ID")
	}
	if len(results) != 3 {
		t.Errorf("expected 3 results, got %d", len(results))
	}

	batch := store.batches[batchID]
	if batch == nil {
		t.Fatal("batch not found in store")
	}
	if batch.Game != "0301" {
		t.Errorf("expected game '0301', got %q", batch.Game)
	}
	if batch.CreatedBy != "test-user" {
		t.Errorf("expected createdBy 'test-user', got %q", batch.CreatedBy)
	}

	extractCount := 0
	parseCount := 0
	for _, w := range store.work {
		switch w.Stage {
		case model.WorkStageExtract:
			extractCount++
		case model.WorkStageParse:
			parseCount++
		}
	}
	if extractCount != 2 {
		t.Errorf("expected 2 extract jobs, got %d", extractCount)
	}
	if parseCount != 1 {
		t.Errorf("expected 1 parse job, got %d", parseCount)
	}
}
