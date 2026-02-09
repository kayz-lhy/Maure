// Package store 提供了 Write-Ahead Log 实现。
//
// WAL（Write-Ahead Log）用于记录索引变更，支持：
//   - 追加记录索引操作（ADD/DELETE）
//   - 支持重放恢复
//   - 支持截断（checkpoint 后清理）
package store

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// WALOperationType 定义 WAL 操作类型。
type WALOperationType string

const (
	// WALOpAdd 表示添加文档操作。
	WALOpAdd WALOperationType = "ADD"
	// WALOpDelete 表示删除文档操作。
	WALOpDelete WALOperationType = "DELETE"
)

// WALOperation 表示一条 WAL 记录。
type WALOperation struct {
	Type      WALOperationType `json:"type"`      // 操作类型
	DocID     int64            `json:"doc_id"`    // 文档 ID
	DocData   []byte           `json:"doc_data"`  // 压缩后的文档数据
	Timestamp int64            `json:"timestamp"` // 时间戳
}

// WAL 是 Write-Ahead Log 的实现。
type WAL struct {
	path       string         // WAL 文件路径
	file       *os.File       // 文件句柄
	codec      *Codec         // 编解码器
	offset     int64          // 当前写入位置
	operations []WALOperation // 内存中的操作缓存
}

// NewWAL 创建新的 WAL。
func NewWAL(path string) (*WAL, error) {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create dir: %w", err)
	}

	// 打开或创建文件
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open wal file: %w", err)
	}

	return &WAL{
		path:       path,
		file:       file,
		codec:      NewCodec(),
		offset:     0,
		operations: make([]WALOperation, 0),
	}, nil
}

// Append 添加一条操作记录。
func (w *WAL) Append(op *WALOperation) error {
	op.Timestamp = time.Now().UnixMilli()

	// 编码操作
	data, err := json.Marshal(op)
	if err != nil {
		return fmt.Errorf("marshal operation: %w", err)
	}

	// 写入长度前缀（4字节大端序）
	length := uint32(len(data))
	lengthBytes := []byte{
		byte(length >> 24),
		byte(length >> 16),
		byte(length >> 8),
		byte(length),
	}

	if _, err := w.file.Write(lengthBytes); err != nil {
		return fmt.Errorf("write length: %w", err)
	}

	if _, err := w.file.Write(data); err != nil {
		return fmt.Errorf("write data: %w", err)
	}

	// 更新偏移量
	w.offset += int64(4 + len(data))

	// 添加到缓存
	w.operations = append(w.operations, *op)

	return nil
}

// AppendAdd 添加文档操作。
func (w *WAL) AppendAdd(docID int64, docData []byte) error {
	// 压缩文档数据
	compressed, err := w.codec.Compress(docData)
	if err != nil {
		return fmt.Errorf("compress doc data: %w", err)
	}

	return w.Append(&WALOperation{
		Type:    WALOpAdd,
		DocID:   docID,
		DocData: compressed,
	})
}

// AppendDelete 添加删除文档操作。
func (w *WAL) AppendDelete(docID int64) error {
	return w.Append(&WALOperation{
		Type:  WALOpDelete,
		DocID: docID,
	})
}

// Read 读取所有操作记录。
func (w *WAL) Read() ([]WALOperation, error) {
	operations := make([]WALOperation, 0)

	// 重置文件位置
	if _, err := w.file.Seek(0, os.SEEK_SET); err != nil {
		return nil, fmt.Errorf("seek: %w", err)
	}

	for {
		// 读取长度前缀
		lengthBytes := make([]byte, 4)
		n, err := w.file.Read(lengthBytes)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read length: %w", err)
		}
		if n != 4 {
			return nil, fmt.Errorf("incomplete length: %d", n)
		}

		length := uint32(lengthBytes[0])<<24 |
			uint32(lengthBytes[1])<<16 |
			uint32(lengthBytes[2])<<8 |
			uint32(lengthBytes[3])

		// 读取数据
		data := make([]byte, length)
		n, err = w.file.Read(data)
		if err != nil {
			return nil, fmt.Errorf("read data: %w", err)
		}
		if int(length) != n {
			return nil, fmt.Errorf("incomplete data: %d/%d", n, length)
		}

		// 解码操作
		var op WALOperation
		if err := json.Unmarshal(data, &op); err != nil {
			return nil, fmt.Errorf("unmarshal operation: %w", err)
		}

		operations = append(operations, op)
	}

	return operations, nil
}

// GetOperations 返回内存中的操作缓存。
func (w *WAL) GetOperations() []WALOperation {
	return w.operations
}

// Truncate 清空 WAL（用于 checkpoint 后）。
func (w *WAL) Truncate() error {
	// 截断文件
	if err := w.file.Truncate(0); err != nil {
		return fmt.Errorf("truncate: %w", err)
	}

	// 重置位置
	if _, err := w.file.Seek(0, os.SEEK_SET); err != nil {
		return fmt.Errorf("seek: %w", err)
	}

	// 清空缓存
	w.operations = make([]WALOperation, 0)
	w.offset = 0

	return nil
}

// Sync 同步到磁盘。
func (w *WAL) Sync() error {
	return w.file.Sync()
}

// Close 关闭 WAL。
func (w *WAL) Close() error {
	// 先同步
	if err := w.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	// 关闭文件
	if err := w.file.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}

	return nil
}

// Path 返回 WAL 文件路径。
func (w *WAL) Path() string {
	return w.path
}

// Size 返回 WAL 文件大小。
func (w *WAL) Size() (int64, error) {
	info, err := w.file.Stat()
	if err != nil {
		return 0, fmt.Errorf("stat: %w", err)
	}
	return info.Size(), nil
}
