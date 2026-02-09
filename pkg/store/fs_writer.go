// Package store 提供了基于文件的索引写入器实现。
package store

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"maure/pkg/document"
	"os"
	"path/filepath"
	"strings"
)

// IndexSnapshot 是索引快照数据。
type IndexSnapshot struct {
	Version     int64                        `json:"version"`      // 格式版本
	DocCount    int64                        `json:"doc_count"`    // 文档数量
	LastDocID   int64                        `json:"last_doc_id"`  // 最后文档 ID
	Documents   map[int64]*document.Document `json:"documents"`    // 文档数据
	Terms       map[string]*Postings         `json:"terms"`        // 倒排表
	FieldLength map[int64]int                `json:"field_length"` // 字段长度
	Checksum    string                       `json:"checksum"`     // 校验和
}

// FsIndexWriter 是基于文件的索引写入器。
type FsIndexWriter struct {
	dir        *FSDirectory   // 目录
	snapshot   *IndexSnapshot // 内存快照
	wal        *WAL           // WAL
	pendingOps int            // 待提交操作数
}

// NewFsIndexWriter 创建新的 FsIndexWriter。
func NewFsIndexWriter(dir *FSDirectory) (*FsIndexWriter, error) {
	// 加载现有快照
	snapshot, err := dir.loadSnapshot()
	if err != nil {
		return nil, fmt.Errorf("load snapshot: %w", err)
	}

	// 获取 WAL
	wal, err := dir.WAL()
	if err != nil {
		return nil, fmt.Errorf("get wal: %w", err)
	}

	writer := &FsIndexWriter{
		dir:        dir,
		snapshot:   snapshot,
		wal:        wal,
		pendingOps: 0,
	}

	// 重放 WAL 中未提交的操作
	if err := writer.replayWAL(); err != nil {
		return nil, fmt.Errorf("replay wal: %w", err)
	}

	return writer, nil
}

// replayWAL 重放 WAL 中的操作。
func (w *FsIndexWriter) replayWAL() error {
	operations, err := w.wal.Read()
	if err != nil {
		return fmt.Errorf("read wal: %w", err)
	}

	for _, op := range operations {
		switch op.Type {
		case WALOpAdd:
			// 解压并解码文档
			docData, err := w.wal.codec.Decompress(op.DocData)
			if err != nil {
				return fmt.Errorf("decompress doc: %w", err)
			}

			var doc document.Document
			if err := gob.NewDecoder(bytes.NewReader(docData)).Decode(&doc); err != nil {
				return fmt.Errorf("decode doc: %w", err)
			}

			// 添加到快照
			if _, exists := w.snapshot.Documents[op.DocID]; !exists {
				w.snapshot.Documents[op.DocID] = &doc
				if w.snapshot.LastDocID < op.DocID {
					w.snapshot.LastDocID = op.DocID
				}
			}

			// 更新倒排索引
			w.updateInvertedIndex(op.DocID, &doc)

			// 更新字段长度
			length := w.calculateFieldLength(&doc)
			w.snapshot.FieldLength[op.DocID] = length

			w.pendingOps++
			w.snapshot.DocCount++

		case WALOpDelete:
			delete(w.snapshot.Documents, op.DocID)
			delete(w.snapshot.FieldLength, op.DocID)
			// 从倒排表移除
			for _, postings := range w.snapshot.Terms {
				for i, id := range postings.DocIDs {
					if id == op.DocID {
						postings.DocIDs = append(postings.DocIDs[:i], postings.DocIDs[i+1:]...)
						postings.Freqs = append(postings.Freqs[:i], postings.Freqs[i+1:]...)
						postings.Positions = append(postings.Positions[:i], postings.Positions[i+1:]...)
						break
					}
				}
			}
			w.pendingOps++
			w.snapshot.DocCount--
		}
	}

	return nil
}

// loadSnapshot 加载现有快照或创建新的。
func (d *FSDirectory) loadSnapshot() (*IndexSnapshot, error) {
	snapshots, err := d.ListSnapshots()
	if err != nil {
		return nil, err
	}

	if len(snapshots) == 0 {
		// 创建新的快照
		return &IndexSnapshot{
			Version:     CurrentVersion,
			Documents:   make(map[int64]*document.Document),
			Terms:       make(map[string]*Postings),
			FieldLength: make(map[int64]int),
		}, nil
	}

	// 加载最新的快照
	latestSnap := snapshots[len(snapshots)-1]
	snapPath := filepath.Join(d.path, latestSnap)

	data, err := os.ReadFile(snapPath)
	if err != nil {
		return nil, fmt.Errorf("read snapshot: %w", err)
	}

	// 验证校验和
	codec := NewCodec()
	checksum := codec.ComputeHash(data)
	if checksum != d.manifest.SnapChecksum {
		// 校验和不匹配，尝试使用旧快照
		if len(snapshots) > 1 {
			latestSnap = snapshots[len(snapshots)-2]
			snapPath = filepath.Join(d.path, latestSnap)
			data, err = os.ReadFile(snapPath)
			if err != nil {
				return nil, fmt.Errorf("read old snapshot: %w", err)
			}
		}
	}

	// 解码快照
	var snapshot IndexSnapshot
	if err := codec.Decode(data, &snapshot); err != nil {
		return nil, fmt.Errorf("decode snapshot: %w", err)
	}

	return &snapshot, nil
}

// AddDocument 添加文档到索引。
func (w *FsIndexWriter) AddDocument(doc *document.Document) (int64, error) {
	docID := w.snapshot.LastDocID + 1

	// 添加到内存快照
	w.snapshot.Documents[docID] = doc
	w.snapshot.LastDocID = docID

	// 计算字段长度（用于评分）
	length := w.calculateFieldLength(doc)
	w.snapshot.FieldLength[docID] = length

	// 更新倒排表
	w.updateInvertedIndex(docID, doc)

	// 编码文档数据
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(doc); err != nil {
		return 0, fmt.Errorf("encode doc: %w", err)
	}

	// 写入 WAL
	if err := w.wal.AppendAdd(docID, buf.Bytes()); err != nil {
		return 0, fmt.Errorf("append wal: %w", err)
	}

	w.pendingOps++

	return docID, nil
}

// calculateFieldLength 计算文档的字段总长度。
func (w *FsIndexWriter) calculateFieldLength(doc *document.Document) int {
	length := 0
	for _, field := range doc.Fields {
		if field.Indexed && field.Tokenized {
			length += len(field.StringValue())
		}
	}
	return length
}

// updateInvertedIndex 更新倒排索引。
func (w *FsIndexWriter) updateInvertedIndex(docID int64, doc *document.Document) {
	for _, field := range doc.Fields {
		if !field.Indexed {
			continue
		}

		// 分词
		tokens := w.tokenize(field)
		for pos, term := range tokens {

			postings, ok := w.snapshot.Terms[term]
			if !ok {
				postings = NewPostings()
				w.snapshot.Terms[term] = postings
			}

			// 检查 docID 是否已存在（避免重复添加）
			exists := false
			for _, id := range postings.DocIDs {
				if id == docID {
					exists = true
					break
				}
			}

			if !exists {
				postings.DocIDs = append(postings.DocIDs, docID)
				postings.Freqs = append(postings.Freqs, 1)
				postings.Positions = append(postings.Positions, []int{pos})
			}
		}
	}
}

// tokenize 对字段进行分词。
func (w *FsIndexWriter) tokenize(field *document.Field) []string {
	// 简单分词：按空格分割并转小写
	text := field.StringValue()
	words := make([]string, 0)
	word := ""
	for _, c := range text {
		if c == ' ' || c == ',' || c == '.' || c == '\t' || c == '\n' {
			if len(word) > 0 {
				words = append(words, strings.ToLower(word))
				word = ""
			}
		} else {
			word += string(c)
		}
	}
	if len(word) > 0 {
		words = append(words, strings.ToLower(word))
	}
	return words
}

// Delete 删除文档。
func (w *FsIndexWriter) Delete(docID int64) error {
	// 从内存快照移除
	delete(w.snapshot.Documents, docID)
	delete(w.snapshot.FieldLength, docID)

	// 从倒排表移除
	for _, postings := range w.snapshot.Terms {
		for i, id := range postings.DocIDs {
			if id == docID {
				postings.DocIDs = append(postings.DocIDs[:i], postings.DocIDs[i+1:]...)
				postings.Freqs = append(postings.Freqs[:i], postings.Freqs[i+1:]...)
				postings.Positions = append(postings.Positions[:i], postings.Positions[i+1:]...)
				break
			}
		}
	}

	// 写入 WAL
	if err := w.wal.AppendDelete(docID); err != nil {
		return fmt.Errorf("append wal: %w", err)
	}

	w.pendingOps++
	return nil
}

// UpdateDocument 更新文档。
func (w *FsIndexWriter) UpdateDocument(docID int64, doc *document.Document) error {
	if err := w.Delete(docID); err != nil {
		return err
	}
	_, err := w.AddDocument(doc)
	return err
}

// Commit 提交更改并生成快照。
func (w *FsIndexWriter) Commit() error {
	if w.pendingOps == 0 {
		return nil
	}

	// 同步 WAL
	if err := w.wal.Sync(); err != nil {
		return fmt.Errorf("sync wal: %w", err)
	}

	// 生成新快照
	snapPath := w.dir.GetSnapshotPath(w.snapshot.LastDocID)

	// 计算文档数量
	w.snapshot.DocCount = int64(len(w.snapshot.Documents))

	// 编码并压缩快照
	encoded, err := w.dir.Codec().Encode(w.snapshot)
	if err != nil {
		return fmt.Errorf("encode snapshot: %w", err)
	}

	// 计算校验和
	checksum := w.dir.Codec().ComputeHash(encoded)

	// 写入快照文件
	if err := os.WriteFile(snapPath, encoded, 0644); err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}

	// 更新 manifest
	dir := w.dir
	walSize, _ := w.wal.Size()
	dir.UpdateManifest(func(m *Manifest) {
		m.SnapPath = filepath.Base(snapPath)
		m.SnapChecksum = checksum
		m.WalOffset = walSize
	})

	// 截断 WAL（已 checkpoint 的操作不再需要）
	if err := w.wal.Truncate(); err != nil {
		return fmt.Errorf("truncate wal: %w", err)
	}

	// 重置待提交计数
	w.pendingOps = 0

	return nil
}

// Close 关闭写入器。
func (w *FsIndexWriter) Close() error {
	// 提交待处理的更改
	if err := w.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}

// PendingOps 返回待提交的操作数。
func (w *FsIndexWriter) PendingOps() int {
	return w.pendingOps
}
