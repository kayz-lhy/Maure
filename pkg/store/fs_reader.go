// Package store 提供了基于文件的索引读取器实现。
package store

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"maure/pkg/document"
)

// ErrDocNotFound 文档未找到。
var ErrDocNotFound = errors.New("document not found")

// FsIndexReader 是基于文件的索引读取器。
type FsIndexReader struct {
	dir      *FSDirectory   // 目录
	snapshot *IndexSnapshot // 快照数据
}

// NewFsIndexReader 创建新的 FsIndexReader。
func NewFsIndexReader(dir *FSDirectory) (*FsIndexReader, error) {
	// 加载快照
	snapshot, err := dir.loadSnapshot()
	if err != nil {
		return nil, fmt.Errorf("load snapshot: %w", err)
	}

	return &FsIndexReader{
		dir:      dir,
		snapshot: snapshot,
	}, nil
}

// DocCount 返回文档数量。
func (r *FsIndexReader) DocCount() int64 {
	return r.snapshot.DocCount
}

// GetDocument 获取指定文档 ID 的文档。
func (r *FsIndexReader) GetDocument(docID int64) (*document.Document, error) {
	doc, ok := r.snapshot.Documents[docID]
	if !ok {
		return nil, ErrDocNotFound
	}
	return doc, nil
}

// Exists 检查文档是否存在。
func (r *FsIndexReader) Exists(docID int64) bool {
	_, ok := r.snapshot.Documents[docID]
	return ok
}

// GetPostings 获取词项的倒排表。
func (r *FsIndexReader) GetPostings(term string) (*Postings, error) {
	p, ok := r.snapshot.Terms[term]
	if !ok {
		return nil, ErrDocNotFound
	}
	return p, nil
}

// GetTerms 获取所有词项。
func (r *FsIndexReader) GetTerms() []string {
	terms := make([]string, 0, len(r.snapshot.Terms))
	for term := range r.snapshot.Terms {
		terms = append(terms, term)
	}
	return terms
}

// GetFieldLength 获取指定文档的字段长度。
func (r *FsIndexReader) GetFieldLength(docID int64) int {
	return r.snapshot.FieldLength[docID]
}

// Close 释放资源。
func (r *FsIndexReader) Close() error {
	return nil
}

// Snapshot 返回快照数据（只读）。
func (r *FsIndexReader) Snapshot() *IndexSnapshot {
	return r.snapshot
}

// ImportDocuments 导入文档数据（用于初始化倒排索引）。
func (r *FsIndexReader) ImportDocuments(idx *IndexData) {
	for docID, doc := range r.snapshot.Documents {
		idx.Documents[docID] = doc
	}

	for term, postings := range r.snapshot.Terms {
		idx.Terms[term] = postings
	}

	for docID, length := range r.snapshot.FieldLength {
		idx.FieldLength[docID] = length
	}

	idx.DocCount = r.snapshot.DocCount
	idx.LastDocID = r.snapshot.LastDocID
}

// IndexData 是索引数据的传输格式。
type IndexData struct {
	Documents   map[int64]*document.Document
	Terms       map[string]*Postings
	FieldLength map[int64]int
	DocCount    int64
	LastDocID   int64
}

// ExportTo 从 IndexData 导出文档和索引数据。
func (i *IndexSnapshot) ExportTo(data *IndexData) {
	for docID, doc := range i.Documents {
		data.Documents[docID] = doc
	}

	for term, postings := range i.Terms {
		data.Terms[term] = postings
	}

	for docID, length := range i.FieldLength {
		data.FieldLength[docID] = length
	}

	data.DocCount = i.DocCount
	data.LastDocID = i.LastDocID
}

// ReplayWAL 重放 WAL 操作。
func (d *IndexData) ReplayWAL(wal *WAL) error {
	operations, err := wal.Read()
	if err != nil {
		return fmt.Errorf("read wal: %w", err)
	}

	for _, op := range operations {
		switch op.Type {
		case WALOpAdd:
			// 解压文档数据
			docData, err := wal.codec.Decompress(op.DocData)
			if err != nil {
				return fmt.Errorf("decompress doc: %w", err)
			}

			// 解码文档
			var doc document.Document
			if err := gob.NewDecoder(bytes.NewReader(docData)).Decode(&doc); err != nil {
				return fmt.Errorf("decode doc: %w", err)
			}

			d.Documents[op.DocID] = &doc

		case WALOpDelete:
			delete(d.Documents, op.DocID)
			delete(d.FieldLength, op.DocID)
		}
	}

	return nil
}
