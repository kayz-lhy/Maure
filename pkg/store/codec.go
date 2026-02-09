// Package store 提供了索引存储的实现。
//
// 支持：
//   - 编码/解码（gob）
//   - 压缩（gzip）
//   - 文件指纹（SHA1）
package store

import (
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"encoding/gob"
	"fmt"
	"io"
	"os"
)

// Codec 负责数据的编码、解码和压缩。
type Codec struct{}

// NewCodec 创建新的 Codec。
func NewCodec() *Codec {
	return &Codec{}
}

// Encode 将数据编码并压缩。
//
// data: 要编码的数据
// 返回: 压缩后的字节切片
func (c *Codec) Encode(data interface{}) ([]byte, error) {
	var buf bytes.Buffer

	// 创建 gzip 写入器
	gw := gzip.NewWriter(&buf)
	defer gw.Close()

	// gob 编码
	enc := gob.NewEncoder(gw)
	if err := enc.Encode(data); err != nil {
		return nil, fmt.Errorf("encode data: %w", err)
	}

	// 确保所有数据写入
	if err := gw.Close(); err != nil {
		return nil, fmt.Errorf("close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// Decode 解码压缩数据。
//
// data: 压缩后的字节切片
// v: 要解码到的目标值指针
func (c *Codec) Decode(data []byte, v interface{}) error {
	// 创建 gzip 读取器
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create gzip reader: %w", err)
	}
	defer gr.Close()

	// gob 解码
	dec := gob.NewDecoder(gr)
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decode data: %w", err)
	}

	return nil
}

// ComputeHash 计算数据的 SHA1 哈希。
func (c *Codec) ComputeHash(data []byte) string {
	h := sha1.New()
	if _, err := h.Write(data); err != nil {
		return ""
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ComputeFileHash 计算文件的 SHA1 哈希。
func (c *Codec) ComputeFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	h := sha1.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", fmt.Errorf("compute hash: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// Compress 压缩数据（仅压缩，不编码）。
func (c *Codec) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	defer gw.Close()

	if _, err := gw.Write(data); err != nil {
		return nil, fmt.Errorf("compress: %w", err)
	}

	if err := gw.Close(); err != nil {
		return nil, fmt.Errorf("close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// Decompress 解压数据（仅解压，不解码）。
func (c *Codec) Decompress(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create gzip reader: %w", err)
	}
	defer gr.Close()

	result, err := io.ReadAll(gr)
	if err != nil {
		return nil, fmt.Errorf("decompress: %w", err)
	}

	return result, nil
}
