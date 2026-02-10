package store

import (
	"os"
	"path/filepath"
	"testing"

	"maure/pkg/document"
)

func TestFSIndexWriter_PersistsAggregatedPostingsAndFieldLength(t *testing.T) {
	dirPath := t.TempDir()
	dir, err := NewFSDirectory(dirPath)
	if err != nil {
		t.Fatalf("new fs directory failed: %v", err)
	}
	defer dir.Close()

	writer, err := dir.CreateIndexWriter()
	if err != nil {
		t.Fatalf("create writer failed: %v", err)
	}

	doc := document.NewDocument()
	doc.Add(document.NewTextField("content", "go go gopher"))
	doc.Add(document.NewStringField("id", "DOC-001"))
	docID, err := writer.AddDocument(doc)
	if err != nil {
		t.Fatalf("add doc failed: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close writer failed: %v", err)
	}

	reader, err := dir.OpenIndexReader()
	if err != nil {
		t.Fatalf("open reader failed: %v", err)
	}
	defer reader.Close()

	fsReader, ok := reader.(*FsIndexReader)
	if !ok {
		t.Fatalf("expected FsIndexReader")
	}
	snap := fsReader.Snapshot()

	goPostings, ok := snap.Terms["go"]
	if !ok {
		t.Fatalf("expected go postings in snapshot")
	}
	if len(goPostings.DocIDs) != 1 {
		t.Fatalf("expected one doc entry for go, got %d", len(goPostings.DocIDs))
	}
	if goPostings.DocIDs[0] != docID {
		t.Fatalf("expected docID %d, got %d", docID, goPostings.DocIDs[0])
	}
	if goPostings.Freqs[0] != 2 {
		t.Fatalf("expected go freq 2, got %d", goPostings.Freqs[0])
	}
	if len(goPostings.Positions[0]) != 2 || goPostings.Positions[0][0] != 0 || goPostings.Positions[0][1] != 1 {
		t.Fatalf("unexpected go positions: %v", goPostings.Positions[0])
	}

	if got := fsReader.GetFieldLength(docID); got != 3 {
		t.Fatalf("expected tokenized field length 3, got %d", got)
	}
}

func TestFSIndexReader_RepairsMissingFieldLengthForOldSnapshot(t *testing.T) {
	dirPath := t.TempDir()
	dir, err := NewFSDirectory(dirPath)
	if err != nil {
		t.Fatalf("new fs directory failed: %v", err)
	}
	defer dir.Close()

	doc := document.NewDocument()
	doc.Add(document.NewTextField("content", "go gopher"))

	legacySnap := &IndexSnapshot{
		Version:   CurrentVersion,
		DocCount:  1,
		LastDocID: 1,
		Documents: map[int64]*document.Document{
			1: doc,
		},
		Terms:       map[string]*Postings{},
		FieldLength: nil,
	}

	encoded, err := dir.Codec().Encode(legacySnap)
	if err != nil {
		t.Fatalf("encode legacy snapshot failed: %v", err)
	}

	snapPath := dir.GetSnapshotPath(1)
	if err := os.WriteFile(snapPath, encoded, 0644); err != nil {
		t.Fatalf("write legacy snapshot failed: %v", err)
	}

	checksum := dir.Codec().ComputeHash(encoded)
	dir.UpdateManifest(func(m *Manifest) {
		m.SnapPath = filepath.Base(snapPath)
		m.SnapChecksum = checksum
		m.LastDocID = 1
	})

	reader, err := dir.OpenIndexReader()
	if err != nil {
		t.Fatalf("open reader failed: %v", err)
	}
	defer reader.Close()

	fsReader, ok := reader.(*FsIndexReader)
	if !ok {
		t.Fatalf("expected FsIndexReader")
	}
	if fsReader.DocCount() != 1 {
		t.Fatalf("expected doc count 1, got %d", fsReader.DocCount())
	}
	if got := fsReader.GetFieldLength(1); got != 2 {
		t.Fatalf("expected repaired field length 2, got %d", got)
	}
}
