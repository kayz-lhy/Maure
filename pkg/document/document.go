// Package document 提供了文档和字段的数据结构。
//
// 文档是搜索引擎的基本存储单元，由多个字段组成。
// 每个字段可以配置不同的属性（是否存储、索引、分词）。
package document

import (
	"strconv"
	"strings"
)

// FieldType 表示字段类型。
type FieldType int

const (
	// FieldTypeText 文本字段，需要分词处理。
	FieldTypeText FieldType = iota
	// FieldTypeString 字符串字段，不分词作为整体存储。
	FieldTypeString
	// FieldTypeInt64 64位整数字段。
	FieldTypeInt64
	// FieldTypeFloat64 64位浮点字段。
	FieldTypeFloat64
	// FieldTypeBool 布尔字段。
	FieldTypeBool
	// FieldTypeDate 日期字段。
	FieldTypeDate
)

// String 返回字段类型的字符串表示。
func (t FieldType) String() string {
	switch t {
	case FieldTypeText:
		return "text"
	case FieldTypeString:
		return "string"
	case FieldTypeInt64:
		return "int64"
	case FieldTypeFloat64:
		return "float64"
	case FieldTypeBool:
		return "bool"
	case FieldTypeDate:
		return "date"
	default:
		return "unknown"
	}
}

// Field 表示文档中的一个字段。
//
// 字段是文档的基本组成单位，包含名称、值和属性配置。
// 通过配置 FieldOptions 可以控制字段的存储、索引和分词行为。
type Field struct {
	Name      string      // 字段名称
	Value     interface{} // 字段值
	FieldType FieldType   // 字段类型

	// 索引选项
	Stored    bool // 是否存储原始值（用于结果返回）
	Indexed   bool // 是否创建索引（用于搜索）
	Tokenized bool // 是否分词（仅对文本字段有效）
}

// NewTextField 创建一个文本字段。
//
// 文本字段会被分词处理，适合搜索内容、标题等。
func NewTextField(name string, value string) *Field {
	return &Field{
		Name:      name,
		Value:     value,
		FieldType: FieldTypeText,
		Indexed:   true,
		Tokenized: true,
		Stored:    true,
	}
}

// NewStringField 创建一个字符串字段。
//
// 字符串字段不分词，适合精确匹配的值如 ID、标签等。
func NewStringField(name string, value string) *Field {
	return &Field{
		Name:      name,
		Value:     value,
		FieldType: FieldTypeString,
		Indexed:   true,
		Tokenized: false,
		Stored:    true,
	}
}

// NewInt64Field 创建一个整数字段。
func NewInt64Field(name string, value int64) *Field {
	return &Field{
		Name:      name,
		Value:     value,
		FieldType: FieldTypeInt64,
		Indexed:   true,
		Tokenized: false,
		Stored:    true,
	}
}

// NewFloat64Field 创建一个浮点数字段。
func NewFloat64Field(name string, value float64) *Field {
	return &Field{
		Name:      name,
		Value:     value,
		FieldType: FieldTypeFloat64,
		Indexed:   true,
		Tokenized: false,
		Stored:    true,
	}
}

// NewStoredField 创建一个仅存储的字段。
//
// 不索引不分词，仅用于存储值供结果返回。
func NewStoredField(name string, value interface{}) *Field {
	return &Field{
		Name:      name,
		Value:     value,
		FieldType: inferType(value),
		Stored:    true,
		Indexed:   false,
		Tokenized: false,
	}
}

// StringValue 返回字段的字符串值。
func (f *Field) StringValue() string {
	switch v := f.Value.(type) {
	case string:
		return v
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return ""
	}
}

// NumberValue 返回字段的数值（int64）。
func (f *Field) NumberValue() int64 {
	switch v := f.Value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

// inferType 推断值的类型。
func inferType(v interface{}) FieldType {
	switch v.(type) {
	case string:
		return FieldTypeString
	case int64:
		return FieldTypeInt64
	case int:
		return FieldTypeInt64
	case float64:
		return FieldTypeFloat64
	case bool:
		return FieldTypeBool
	default:
		return FieldTypeString
	}
}

// Document 表示一个可索引的文档。
//
// 文档是搜索引擎存储和检索的基本单元。
// 每个文档由多个字段组成，可以添加任意数量的字段。
type Document struct {
	Fields []*Field // 字段列表
	Boost  float32  // 文档权重（影响评分）
	id     string   // 内部文档ID
}

// NewDocument 创建一个新的空文档。
func NewDocument() *Document {
	return &Document{
		Fields: make([]*Field, 0),
		Boost:  1.0,
	}
}

// Add 添加一个字段到文档。
func (d *Document) Add(field *Field) {
	d.Fields = append(d.Fields, field)
}

// Get 获取指定名称的字段值。
//
// 返回第一个匹配名称的字段值。
// 如果不存在返回 nil。
func (d *Document) Get(name string) *Field {
	for _, f := range d.Fields {
		if f.Name == name {
			return f
		}
	}
	return nil
}

// GetAll 获取指定名称的所有字段值。
func (d *Document) GetAll(name string) []*Field {
	if len(d.Fields) == 0 {
		return nil
	}
	result := make([]*Field, 0, 4)
	for _, f := range d.Fields {
		if f.Name == name {
			result = append(result, f)
		}
	}
	return result
}

// Values 返回所有字段值的切片。
func (d *Document) Values() []interface{} {
	values := make([]interface{}, len(d.Fields))
	for i, f := range d.Fields {
		values[i] = f.Value
	}
	return values
}

// StringValues 返回所有文本字段值的切片。
func (d *Document) StringValues() []string {
	if len(d.Fields) == 0 {
		return nil
	}
	values := make([]string, 0, len(d.Fields)/2)
	for _, f := range d.Fields {
		if f.FieldType == FieldTypeText || f.FieldType == FieldTypeString {
			values = append(values, f.StringValue())
		}
	}
	return values
}

// SetID 设置文档的内部ID。
func (d *Document) SetID(id string) {
	d.id = id
}

// ID 获取文档的内部ID。
func (d *Document) ID() string {
	return d.id
}

// SetBoost 设置文档权重。
func (d *Document) SetBoost(boost float32) {
	d.Boost = boost
}

// Boost 获取文档权重。
func (d *Document) BoostValue() float32 {
	return d.Boost
}

// String 返回文档的字符串表示（调试用）。
func (d *Document) String() string {
	var sb strings.Builder
	sb.WriteString("Document{")
	for i, f := range d.Fields {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(f.Name)
		sb.WriteString(":")
		sb.WriteString(f.StringValue())
	}
	sb.WriteString("}")
	return sb.String()
}

// NewDocumentWithValues 从字段映射创建文档。
//
// 这是一个便捷函数，用于快速创建带有多个字段的文档。
// values 参数的 key 是字段名，value 可以是 string、int64、float64。
func NewDocumentWithValues(id string, values map[string]interface{}) *Document {
	doc := NewDocument()
	doc.SetID(id)
	for name, value := range values {
		switch v := value.(type) {
		case string:
			doc.Add(NewTextField(name, v))
		case int:
			doc.Add(NewInt64Field(name, int64(v)))
		case int64:
			doc.Add(NewInt64Field(name, v))
		case float64:
			doc.Add(NewFloat64Field(name, v))
		case bool:
			doc.Add(NewStoredField(name, v))
		default:
			doc.Add(NewStoredField(name, v))
		}
	}
	return doc
}
