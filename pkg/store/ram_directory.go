package store

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"maure/pkg/document"
	"sync"
)

var (
	// ErrFileNotFound 文件不存在。
	ErrFileNotFound = errors.New("file not found")
	// ErrClosed 已关闭。
	ErrClosed = errors.New("directory closed")
)

// RAMDirectory 是基于内存的目录实现。
//
// RAMDirectory 将所有索引数据存储在内存中，
// 适用于小规模索引或测试场景。
type RAMDirectory struct {
	mu     sync.RWMutex
	files  map[string][]byte
	closed bool
}

// NewRAMDirectory 创建新的 RAMDirectory。
func NewRAMDirectory() *RAMDirectory {
	return &RAMDirectory{
		files: make(map[string][]byte),
	}
}

// CreateIndexWriter 实现了 Directory 接口。
func (d *RAMDirectory) CreateIndexWriter() (IndexWriter, error) {
	if d.closed {
		return nil, ErrClosed
	}
	return NewRAMIndexWriter(d), nil
}

// OpenIndexWriter 实现了 Directory 接口。
func (d *RAMDirectory) OpenIndexWriter() (IndexWriter, error) {
	return d.CreateIndexWriter()
}

// OpenIndexReader 实现了 Directory 接口。
func (d *RAMDirectory) OpenIndexReader() (IndexReader, error) {
	if d.closed {
		return nil, ErrClosed
	}
	return NewRAMIndexReader(d), nil
}

// Exists 实现了 Directory 接口。
func (d *RAMDirectory) Exists(name string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, ok := d.files[name]
	return ok
}

// DeleteFile 实现了 Directory 接口。
func (d *RAMDirectory) DeleteFile(name string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.files, name)
	return nil
}

// ListFiles 实现了 Directory 接口。
func (d *RAMDirectory) ListFiles() ([]string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	files := make([]string, 0, len(d.files))
	for name := range d.files {
		files = append(files, name)
	}
	return files, nil
}

// Close 实现了 Directory 接口。
func (d *RAMDirectory) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	d.files = nil
	return nil
}

// readFile 读取文件内容（内部使用）。
func (d *RAMDirectory) readFile(name string) ([]byte, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	data, ok := d.files[name]
	if !ok {
		return nil, ErrFileNotFound
	}
	return data, nil
}

// writeFile 写入文件内容（内部使用）。
func (d *RAMDirectory) writeFile(name string, data []byte) error {
	if d.closed {
		return ErrClosed
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.files[name] = data
	return nil
}

// RAMIndexWriter 是基于 RAM 的索引写入器。
type RAMIndexWriter struct {
	dir    *RAMDirectory
	mu     sync.Mutex
	closed bool
}

// NewRAMIndexWriter 创建新的 RAMIndexWriter。
func NewRAMIndexWriter(dir *RAMDirectory) *RAMIndexWriter {
	return &RAMIndexWriter{dir: dir}
}

// AddDocument 实现了 IndexWriter 接口。
func (w *RAMIndexWriter) AddDocument(doc *document.Document) (int64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return 0, ErrClosed
	}

	// 分配文档 ID（从 segments 文件读取计数器）
	docID := int64(1)
	segmentsData, err := w.dir.readFile("segments")
	if err == nil {
		docID = int64(binary.BigEndian.Uint64(segmentsData)) + 1
	}

	// 序列化文档并存储
	var buf bytes.Buffer
	if err := documentWrite(&buf, doc); err != nil {
		return 0, err
	}
	docName := "doc-" + itoa(docID)
	if err := w.dir.writeFile(docName, buf.Bytes()); err != nil {
		return 0, err
	}

	// 更新 segments 文件
	segBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(segBuf, uint64(docID))
	if err := w.dir.writeFile("segments", segBuf); err != nil {
		return 0, err
	}

	return docID, nil
}

// Delete 实现了 IndexWriter 接口。
func (w *RAMIndexWriter) Delete(docID int64) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return ErrClosed
	}

	docName := "doc-" + itoa(docID)
	return w.dir.DeleteFile(docName)
}

// UpdateDocument 实现了 IndexWriter 接口。
func (w *RAMIndexWriter) UpdateDocument(docID int64, doc *document.Document) error {
	if err := w.Delete(docID); err != nil {
		return err
	}
	_, err := w.AddDocument(doc)
	return err
}

// Commit 实现了 IndexWriter 接口。
func (w *RAMIndexWriter) Commit() error {
	// RAMDirectory 无需额外提交
	return nil
}

// Close 实现了 IndexWriter 接口。
func (w *RAMIndexWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	return nil
}

// RAMIndexReader 是基于 RAM 的索引读取器。
type RAMIndexReader struct {
	dir    *RAMDirectory
	mu     sync.RWMutex
	closed bool
}

// NewRAMIndexReader 创建新的 RAMIndexReader。
func NewRAMIndexReader(dir *RAMDirectory) *RAMIndexReader {
	return &RAMIndexReader{dir: dir}
}

// DocCount 实现了 IndexReader 接口。
func (r *RAMIndexReader) DocCount() int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	segmentsData, err := r.dir.readFile("segments")
	if err != nil {
		return 0
	}
	return int64(binary.BigEndian.Uint64(segmentsData))
}

// GetDocument 实现了 IndexReader 接口。
func (r *RAMIndexReader) GetDocument(docID int64) (*document.Document, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	docName := "doc-" + itoa(docID)
	data, err := r.dir.readFile(docName)
	if err != nil {
		return nil, err
	}
	return documentRead(bytes.NewReader(data))
}

// Exists 实现了 IndexReader 接口。
func (r *RAMIndexReader) Exists(docID int64) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	docName := "doc-" + itoa(docID)
	return r.dir.Exists(docName)
}

// Close 实现了 IndexReader 接口。
func (r *RAMIndexReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return nil
}

// RAMIndexOutput 是基于内存的索引输出。
type RAMIndexOutput struct {
	buf bytes.Buffer
}

// Write 实现了 io.Writer 接口。
func (o *RAMIndexOutput) Write(p []byte) (n int, err error) {
	return o.buf.Write(p)
}

// WriteVInt 写入可变长度整数。
func (o *RAMIndexOutput) WriteVInt(v int64) error {
	// 简单实现：固定 8 字节
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(v))
	_, err := o.buf.Write(buf)
	return err
}

// WriteString 写入字符串。
func (o *RAMIndexOutput) WriteString(s string) error {
	// 写入长度 + 内容
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(s)))
	if _, err := o.buf.Write(lenBuf); err != nil {
		return err
	}
	_, err := o.buf.WriteString(s)
	return err
}

// Flush 刷新到存储（由调用者处理）。
func (o *RAMIndexOutput) Flush() error {
	return nil
}

// Close 关闭输出流。
func (o *RAMIndexOutput) Close() error {
	return nil
}

// Bytes 返回写入的数据。
func (o *RAMIndexOutput) Bytes() []byte {
	return o.buf.Bytes()
}

// RAMIndexInput 是基于内存的索引输入。
type RAMIndexInput struct {
	data  []byte
	pos   int
	limit int
}

// NewRAMIndexInput 从字节切片创建输入。
func NewRAMIndexInput(data []byte) *RAMIndexInput {
	return &RAMIndexInput{
		data:  data,
		limit: len(data),
	}
}

// Read 实现了 io.Reader 接口。
func (i *RAMIndexInput) Read(p []byte) (n int, err error) {
	if i.pos >= i.limit {
		return 0, io.EOF
	}
	n = copy(p, i.data[i.pos:i.limit])
	i.pos += n
	return n, nil
}

// ReadVInt 读取可变长度整数。
func (i *RAMIndexInput) ReadVInt() (int64, error) {
	if i.pos+8 > i.limit {
		return 0, io.EOF
	}
	v := int64(binary.BigEndian.Uint64(i.data[i.pos : i.pos+8]))
	i.pos += 8
	return v, nil
}

// ReadString 读取字符串。
func (i *RAMIndexInput) ReadString() (string, error) {
	if i.pos+4 > i.limit {
		return "", io.EOF
	}
	length := int(binary.BigEndian.Uint32(i.data[i.pos : i.pos+4]))
	i.pos += 4
	if i.pos+length > i.limit {
		return "", io.EOF
	}
	s := string(i.data[i.pos : i.pos+length])
	i.pos += length
	return s, nil
}

// Seek 移动读取位置。
func (i *RAMIndexInput) Seek(offset int64, whence int) error {
	var newPos int
	switch whence {
	case io.SeekStart:
		newPos = int(offset)
	case io.SeekCurrent:
		newPos = i.pos + int(offset)
	case io.SeekEnd:
		newPos = i.limit + int(offset)
	default:
		return errors.New("invalid whence")
	}
	if newPos < 0 || newPos > i.limit {
		return errors.New("seek out of range")
	}
	i.pos = newPos
	return nil
}

// Close 关闭输入流。
func (i *RAMIndexInput) Close() error {
	return nil
}

// RAMPostingsWriter 是基于内存的倒排表写入器。
type RAMPostingsWriter struct {
	postings map[string]*Postings
	mu       sync.Mutex
}

// NewRAMPostingsWriter 创建新的 RAMPostingsWriter。
func NewRAMPostingsWriter() *RAMPostingsWriter {
	return &RAMPostingsWriter{
		postings: make(map[string]*Postings),
	}
}

// AddPostings 添加倒排表条目。
func (w *RAMPostingsWriter) AddPostings(term string, docID int64, freq int, positions []int) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	p, ok := w.postings[term]
	if !ok {
		p = NewPostings()
		w.postings[term] = p
	}
	p.DocIDs = append(p.DocIDs, docID)
	p.Freqs = append(p.Freqs, freq)
	p.Positions = append(p.Positions, positions)
	return nil
}

// Finish 完成写入。
func (w *RAMPostingsWriter) Finish() error {
	return nil
}

// Close 释放资源。
func (w *RAMPostingsWriter) Close() error {
	w.postings = nil
	return nil
}

// GetPostings 获取倒排表。
func (w *RAMPostingsWriter) GetPostings(term string) *Postings {
	return w.postings[term]
}

// RAMPostingsReader 是基于内存的倒排表读取器。
type RAMPostingsReader struct {
	postings map[string]*Postings
}

// NewRAMPostingsReader 创建新的 RAMPostingsReader。
func NewRAMPostingsReader() *RAMPostingsReader {
	return &RAMPostingsReader{
		postings: make(map[string]*Postings),
	}
}

// GetPostings 获取词项的倒排表。
func (r *RAMPostingsReader) GetPostings(term string) (*Postings, error) {
	return r.postings[term], nil
}

// Close 释放资源。
func (r *RAMPostingsReader) Close() error {
	return nil
}

// SetPostings 设置倒排表（用于内部初始化）。
func (r *RAMPostingsReader) SetPostings(postings map[string]*Postings) {
	r.postings = postings
}

// documentWrite 序列化文档。
func documentWrite(w io.Writer, doc *document.Document) error {
	// 写入字段数量
	if err := binary.Write(w, binary.BigEndian, int32(len(doc.Fields))); err != nil {
		return err
	}
	// 写入每个字段
	for _, f := range doc.Fields {
		// 字段名长度 + 内容
		nameBytes := []byte(f.Name)
		if err := binary.Write(w, binary.BigEndian, int32(len(nameBytes))); err != nil {
			return err
		}
		if _, err := w.Write(nameBytes); err != nil {
			return err
		}
		// 字段类型
		if err := binary.Write(w, binary.BigEndian, int32(f.FieldType)); err != nil {
			return err
		}
		// 值
		switch v := f.Value.(type) {
		case string:
			valBytes := []byte(v)
			if err := binary.Write(w, binary.BigEndian, int32(len(valBytes))); err != nil {
				return err
			}
			if _, err := w.Write(valBytes); err != nil {
				return err
			}
		case int64:
			if err := binary.Write(w, binary.BigEndian, v); err != nil {
				return err
			}
		case float64:
			if err := binary.Write(w, binary.BigEndian, v); err != nil {
				return err
			}
		case bool:
			if err := binary.Write(w, binary.BigEndian, v); err != nil {
				return err
			}
		}
	}
	// 写入 Boost
	if err := binary.Write(w, binary.BigEndian, doc.Boost); err != nil {
		return err
	}
	return nil
}

// documentRead 反序列化文档。
func documentRead(r io.Reader) (*document.Document, error) {
	var fieldCount int32
	if err := binary.Read(r, binary.BigEndian, &fieldCount); err != nil {
		return nil, err
	}
	doc := document.NewDocument()
	for i := 0; i < int(fieldCount); i++ {
		var nameLen int32
		if err := binary.Read(r, binary.BigEndian, &nameLen); err != nil {
			return nil, err
		}
		nameBytes := make([]byte, nameLen)
		if _, err := io.ReadFull(r, nameBytes); err != nil {
			return nil, err
		}
		var fieldType int32
		if err := binary.Read(r, binary.BigEndian, &fieldType); err != nil {
			return nil, err
		}
		// 简化：只读取字符串值
		var valLen int32
		if err := binary.Read(r, binary.BigEndian, &valLen); err != nil {
			return nil, err
		}
		valBytes := make([]byte, valLen)
		if _, err := io.ReadFull(r, valBytes); err != nil {
			return nil, err
		}
		doc.Add(document.NewStoredField(string(nameBytes), string(valBytes)))
	}
	var boost float32
	if err := binary.Read(r, binary.BigEndian, &boost); err != nil {
		return nil, err
	}
	doc.SetBoost(boost)
	return doc, nil
}

// itoa 将 int64 转换为字符串。
func itoa(n int64) string {
	buf := make([]byte, 20)
	neg := n < 0
	if neg {
		n = -n
	}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if i == len(buf) {
		i--
		buf[i] = '0'
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// RAMBuffer 是基于内存的文件缓冲区。
type RAMBuffer struct {
	buf bytes.Buffer
	w   *bufio.Writer
}

// NewRAMBuffer 创建新的 RAMBuffer。
func NewRAMBuffer() *RAMBuffer {
	return &RAMBuffer{
		w: bufio.NewWriter(nil),
	}
}

// Write 实现了 io.Writer 接口。
func (b *RAMBuffer) Write(p []byte) (n int, err error) {
	return b.buf.Write(p)
}

// Flush 刷新到内部缓冲区。
func (b *RAMBuffer) Flush() error {
	return b.w.Flush()
}

// Bytes 返回缓冲区的内容。
func (b *RAMBuffer) Bytes() []byte {
	return b.buf.Bytes()
}
