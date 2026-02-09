// Package store 提供了基于文件的目录实现。
//
// FSDirectory 支持：
//   - 索引文件管理
//   - 文件锁防止并发写入
//   - 增量 checkpoint
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FSDirectory 是基于文件的目录实现。
type FSDirectory struct {
	path     string       // 根目录路径
	codec    *Codec       // 编解码器
	mu       sync.RWMutex // 读写锁
	manifest *Manifest    // 元数据
	wal      *WAL         // WAL
	closed   bool
}

// Manifest 是索引元数据。
type Manifest struct {
	Version      int64  `json:"version"`       // 格式版本
	SnapPath     string `json:"snap_path"`     // 快照文件路径
	SnapChecksum string `json:"snap_checksum"` // 快照校验和
	WalOffset    int64  `json:"wal_offset"`    // WAL 位置
	LastDocID    int64  `json:"last_doc_id"`   // 最后文档 ID
	CreatedAt    int64  `json:"created_at"`    // 创建时间
	UpdatedAt    int64  `json:"updated_at"`    // 更新时间
}

// NewFSDirectory 创建新的 FSDirectory。
func NewFSDirectory(path string) (*FSDirectory, error) {
	// 确保目录存在
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("create directory: %w", err)
	}

	dir := &FSDirectory{
		path:  path,
		codec: NewCodec(),
	}

	// 加载或创建 manifest
	if err := dir.loadManifest(); err != nil {
		return nil, fmt.Errorf("load manifest: %w", err)
	}

	return dir, nil
}

// loadManifest 加载或创建 manifest。
func (d *FSDirectory) loadManifest() error {
	manifestPath := filepath.Join(d.path, "manifest.json")

	// 尝试读取现有 manifest
	data, err := os.ReadFile(manifestPath)
	if err == nil {
		var manifest Manifest
		if err := json.Unmarshal(data, &manifest); err == nil {
			d.manifest = &manifest
			return nil
		}
	}

	// 创建新的 manifest
	d.manifest = &Manifest{
		Version:   CurrentVersion,
		CreatedAt: time.Now().UnixMilli(),
		UpdatedAt: time.Now().UnixMilli(),
	}

	return d.saveManifest()
}

// saveManifest 保存 manifest。
func (d *FSDirectory) saveManifest() error {
	data, err := json.MarshalIndent(d.manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	manifestPath := filepath.Join(d.path, "manifest.json")
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

// UpdateManifest 更新 manifest 并保存。
func (d *FSDirectory) UpdateManifest(fn func(*Manifest)) {
	d.manifest.UpdatedAt = time.Now().UnixMilli()
	fn(d.manifest)
	d.saveManifest()
}

// CreateIndexWriter 创建索引写入器。
func (d *FSDirectory) CreateIndexWriter() (IndexWriter, error) {
	return NewFsIndexWriter(d)
}

// OpenIndexWriter 打开已存在的索引写入器。
func (d *FSDirectory) OpenIndexWriter() (IndexWriter, error) {
	return NewFsIndexWriter(d)
}

// OpenIndexReader 打开索引读取器。
func (d *FSDirectory) OpenIndexReader() (IndexReader, error) {
	return NewFsIndexReader(d)
}

// Exists 检查文件是否存在。
func (d *FSDirectory) Exists(name string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	path := filepath.Join(d.path, name)
	_, err := os.Stat(path)
	return err == nil
}

// DeleteFile 删除文件。
func (d *FSDirectory) DeleteFile(name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	path := filepath.Join(d.path, name)
	return os.Remove(path)
}

// ListFiles 列出所有文件。
func (d *FSDirectory) ListFiles() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	entries, err := os.ReadDir(d.path)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, entry.Name())
	}

	return files, nil
}

// ListSnapshots 列出所有快照文件。
func (d *FSDirectory) ListSnapshots() ([]string, error) {
	files, err := d.ListFiles()
	if err != nil {
		return nil, err
	}

	snapshots := make([]string, 0)
	for _, f := range files {
		if hasPrefix(f, "snapshot_") && hasSuffix(f, ".dat.gz") {
			snapshots = append(snapshots, f)
		}
	}

	return snapshots, nil
}

// ListWALFiles 列出所有 WAL 文件。
func (d *FSDirectory) ListWALFiles() ([]string, error) {
	files, err := d.ListFiles()
	if err != nil {
		return nil, err
	}

	walFiles := make([]string, 0)
	for _, f := range files {
		if hasPrefix(f, "wal_") && hasSuffix(f, ".log") {
			walFiles = append(walFiles, f)
		}
	}

	return walFiles, nil
}

// Close 释放资源。
func (d *FSDirectory) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil
	}

	d.closed = true

	// 关闭 WAL
	if d.wal != nil {
		d.wal.Close()
	}

	return nil
}

// Path 返回目录路径。
func (d *FSDirectory) Path() string {
	return d.path
}

// Codec 返回编解码器。
func (d *FSDirectory) Codec() *Codec {
	return d.codec
}

// Manifest 返回元数据。
func (d *FSDirectory) Manifest() *Manifest {
	return d.manifest
}

// WAL 返回 WAL（打开已存在的或创建新的）。
func (d *FSDirectory) WAL() (*WAL, error) {
	if d.wal != nil {
		return d.wal, nil
	}

	// 查找或创建 WAL 文件
	walFiles, err := d.ListWALFiles()
	if err != nil {
		return nil, err
	}

	var walPath string
	if len(walFiles) > 0 {
		// 使用最新的 WAL 文件
		walPath = filepath.Join(d.path, walFiles[len(walFiles)-1])
	} else {
		// 创建新的 WAL 文件
		walPath = filepath.Join(d.path, fmt.Sprintf("wal_%03d.log", 1))
	}

	wal, err := NewWAL(walPath)
	if err != nil {
		return nil, err
	}

	d.wal = wal
	return wal, nil
}

// GetSnapshotPath 获取快照文件路径。
func (d *FSDirectory) GetSnapshotPath(version int64) string {
	return filepath.Join(d.path, fmt.Sprintf("snapshot_%03d.dat.gz", version))
}

// CurrentVersion 当前版本号。
const CurrentVersion = 1

// hasPrefix 检查字符串是否有指定前缀。
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// hasSuffix 检查字符串是否有指定后缀。
func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
