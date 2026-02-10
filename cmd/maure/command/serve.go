package command

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"maure/pkg/document"
	"maure/pkg/index"
	"maure/pkg/query"
)

// ServeCommand HTTP 服务命令。
type ServeCommand struct {
	*BaseCommand
	port int
}

// NewServeCommand 创建服务命令。
func NewServeCommand() *ServeCommand {
	cmd := &ServeCommand{
		BaseCommand: NewBaseCommand("serve", "maure serve", "启动 HTTP API 服务"),
	}
	cmd.desc = "启动 HTTP API 服务"
	cmd.flags.IntVar(&cmd.port, "port", 8080, "服务端口")
	cmd.flags = flag.NewFlagSet("serve", flag.ContinueOnError)
	cmd.flags.IntVar(&cmd.port, "port", 8080, "服务端口")
	return cmd
}

// Execute 启动服务。
func (c *ServeCommand) Execute(args []string, opts GlobalOptions) error {
	path := opts.IndexPath
	if len(args) >= 1 {
		path = args[0]
	}
	if path == "" {
		path = "."
	}

	ctx, err := NewIndexContext(path, opts)
	if err != nil {
		ExitWithError(fmt.Errorf("打开索引失败: %w", err))
	}

	ramIdx := index.NewRAMIndex(ctx.Analyzer)
	reader := ctx.Reader
	for i := int64(1); i <= reader.DocCount(); i++ {
		doc, err := reader.GetDocument(i)
		if err != nil {
			continue
		}
		if _, err := ramIdx.Add(doc); err != nil {
			continue
		}
	}

	server := &Server{
		idx:    ramIdx,
		ctx:    ctx,
		parser: query.NewQueryParser(),
		port:   c.port,
	}

	fmt.Printf("启动服务: http://localhost:%d\n", c.port)
	fmt.Printf("索引目录: %s\n", path)
	fmt.Println("\nAPI 端点:")
	fmt.Println("  GET  /            - API 信息")
	fmt.Println("  GET  /search?q=   - 搜索文档")
	fmt.Println("  GET  /doc/:id     - 获取文档")
	fmt.Println("  GET  /stats       - 索引统计")
	fmt.Println("  POST /add         - 添加文档")
	fmt.Println("  DELETE /doc/:id   - 删除文档")
	fmt.Println("\n按 Ctrl+C 停止服务")

	return server.Start()
}

// Server HTTP 服务器。
type Server struct {
	idx    *index.RAMIndex
	ctx    *IndexContext
	parser *query.QueryParser
	port   int
}

// Start 启动服务器。
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/search", s.handleSearch)
	mux.HandleFunc("/doc/", s.handleDoc)
	mux.HandleFunc("/stats", s.handleStats)
	mux.HandleFunc("/add", s.handleAdd)
	mux.HandleFunc("/delete", s.handleDelete)

	return http.ListenAndServe(":"+strconv.Itoa(s.port), mux)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"name":    "Maure Search Engine",
		"version": Version,
		"endpoints": map[string]string{
			"search": "/search?q=<query>",
			"doc":    "/doc/<id>",
			"stats":  "/stats",
			"add":    "POST /add",
			"delete": "DELETE /doc/<id>",
		},
	}); err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		http.Error(w, "缺少查询参数 q", http.StatusBadRequest)
		return
	}

	parsedQuery, err := s.parser.Parse(q)
	if err != nil {
		http.Error(w, "解析查询失败: "+err.Error(), http.StatusBadRequest)
		return
	}

	results, err := s.idx.Search(parsedQuery, 20)
	if err != nil {
		http.Error(w, "搜索失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	type Result struct {
		DocID int64   `json:"doc_id"`
		Score float32 `json:"score"`
	}

	response := make([]Result, len(results))
	for i, r := range results {
		response[i] = Result{DocID: r.DocID, Score: r.Score}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}

func (s *Server) handleDoc(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/doc/"):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "无效的文档 ID", http.StatusBadRequest)
		return
	}

	doc, err := s.ctx.Reader.GetDocument(id)
	if err != nil {
		http.Error(w, "文档不存在", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(doc); err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"doc_count": s.idx.DocCount(),
		"num_terms": s.idx.Inverted().NumTerms(),
	}); err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}

func (s *Server) handleAdd(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "读取请求失败", http.StatusBadRequest)
		return
	}

	var req struct {
		ID     string                 `json:"id"`
		Fields map[string]interface{} `json:"fields"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "解析请求失败", http.StatusBadRequest)
		return
	}

	doc := document.NewDocument()
	doc.SetID(req.ID)
	for name, value := range req.Fields {
		switch v := value.(type) {
		case string:
			doc.Add(document.NewTextField(name, v))
		case int:
			doc.Add(document.NewInt64Field(name, int64(v)))
		case float64:
			doc.Add(document.NewFloat64Field(name, v))
		}
	}

	docID, err := s.idx.Add(doc)
	if err != nil {
		http.Error(w, "添加失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"doc_id": docID,
		"status": "success",
	}); err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		http.Error(w, "缺少文档 ID", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "无效的文档 ID", http.StatusBadRequest)
		return
	}

	if err := s.idx.Delete(id); err != nil {
		http.Error(w, "删除失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
	}); err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}

func init() {
	RegisterCommand(NewServeCommand())
}
