package serve

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/tamnd/olymmath-agent/site"
)

// The versions of the MCP protocol that the server speaks, the newest first.
var protocols = []string{"2025-06-18", "2025-03-26", "2024-11-05"}

// rpcRequest is a JSON-RPC 2.0 request or notification. A notification has no id.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// Tool is one tool of the MCP server.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func object(required []string, props map[string]any) map[string]any {
	o := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		o["required"] = required
	}
	return o
}

var (
	str = func(d string) map[string]any { return map[string]any{"type": "string", "description": d} }
	num = func(d string) map[string]any { return map[string]any{"type": "integer", "description": d} }
)

// Tools are the tools of the MCP server. They give the same records as the HTTP methods.
var Tools = []Tool{
	{"get_problem", "Get a problem of olymmath by its tag: the statement, its source and the facts. The answer and the solutions are not in it; use get_solutions for them.", object([]string{"tag"}, map[string]any{"tag": str("the tag of the problem, for example 0HVHSDNNMY")})},
	{"get_solutions", "Get the answer and the solutions of a problem by its tag. Call it only when the user asks for the answer or a solution.", object([]string{"tag"}, map[string]any{"tag": str("the tag of the problem")})},
	{"search_problems", "Search the problems by words of the statement, the title, the locator (for example imo/2024/p1) or the tag.", object([]string{"query"}, map[string]any{"query": str("the words to find"), "limit": num(fmt.Sprintf("the most results, at most %d; 20 when it is not given", MaxResults))})},
	{"list_edition", "List the papers and the problems of one edition of a competition.", object([]string{"competition", "edition"}, map[string]any{"competition": str("the code of the competition, for example imo"), "edition": str("the edition, usually the year, for example 2024")})},
	{"random_problem", "Get a random problem that passes the filters, for practice. The answer and the solutions are not in it.", object(nil, map[string]any{
		"competition": str("the code of the competition"), "topic": str("a topic such as N, or a subtopic such as N.floor"), "type": str("proof or answer"),
		"band_min": num("the lowest difficulty band, 1 to 10"), "band_max": num("the highest difficulty band, 1 to 10"),
		"year_from": num("the first year"), "year_to": num("the last year"), "seed": num("a seed other than 0 gives the same problem each time"),
	})},
}

// toolArgs are the arguments of each tool.
type toolArgs struct {
	Tag         string `json:"tag"`
	Query       string `json:"query"`
	Limit       int    `json:"limit"`
	Competition string `json:"competition"`
	Edition     string `json:"edition"`
	Topic       string `json:"topic"`
	Type        string `json:"type"`
	BandMin     int    `json:"band_min"`
	BandMax     int    `json:"band_max"`
	YearFrom    int    `json:"year_from"`
	YearTo      int    `json:"year_to"`
	Seed        uint64 `json:"seed"`
}

// Call runs a tool and gives its record. The error is for the caller of the tool, such as a tag that no problem has.
func (s *Server) Call(name string, raw json.RawMessage) (any, error) {
	var a toolArgs
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, fmt.Errorf("the arguments are not valid: %w", err)
		}
	}
	switch name {
	case "get_problem", "get_solutions":
		p := s.api.Lookup(a.Tag)
		if p == nil {
			return nil, fmt.Errorf("no problem has the tag %q", a.Tag)
		}
		if name == "get_problem" {
			return s.api.Problem(p), nil
		}
		return s.api.Solutions(p), nil
	case "search_problems":
		return s.Search(a.Query, a.Limit), nil
	case "list_edition":
		e := s.api.Edition(a.Competition, a.Edition)
		if e == nil {
			return nil, fmt.Errorf("no edition %s/%s", a.Competition, a.Edition)
		}
		return s.api.EditionRecord(e), nil
	case "random_problem":
		f := Filter{Competition: a.Competition, Topic: a.Topic, Type: a.Type, BandMin: a.BandMin, BandMax: a.BandMax, YearFrom: a.YearFrom, YearTo: a.YearTo}
		if p, ok := s.Random(f, a.Seed); ok {
			return p, nil
		}
		return nil, errors.New("no problem passes the filters")
	}
	return nil, fmt.Errorf("no tool %q", name)
}

// handle answers one JSON-RPC message. It gives nil for a notification.
func (s *Server) handle(msg []byte) *rpcResponse {
	var req rpcRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return &rpcResponse{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &rpcError{-32700, "parse error"}}
	}
	if len(req.ID) == 0 {
		return nil
	}
	res := &rpcResponse{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &p)
		version := protocols[0]
		if slices.Contains(protocols, p.ProtocolVersion) {
			version = p.ProtocolVersion
		}
		res.Result = map[string]any{
			"protocolVersion": version,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "olymmath", "version": s.version},
			"instructions":    "The tools read the problems of olymmath. get_problem and random_problem hide the answer and the solutions; get_solutions gives them.",
		}
	case "ping":
		res.Result = map[string]any{}
	case "tools/list":
		res.Result = map[string]any{"tools": Tools}
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			res.Error = &rpcError{-32602, "invalid params"}
			return res
		}
		if !slices.ContainsFunc(Tools, func(t Tool) bool { return t.Name == p.Name }) {
			res.Error = &rpcError{-32602, fmt.Sprintf("no tool %q", p.Name)}
			return res
		}
		v, err := s.Call(p.Name, p.Arguments)
		if err != nil {
			res.Result = map[string]any{"content": []map[string]any{{"type": "text", "text": err.Error()}}, "isError": true}
			return res
		}
		text, err := site.Marshal(v)
		if err != nil {
			res.Error = &rpcError{-32603, err.Error()}
			return res
		}
		res.Result = map[string]any{"content": []map[string]any{{"type": "text", "text": string(text)}}, "structuredContent": v}
	default:
		res.Error = &rpcError{-32601, "method not found"}
	}
	return res
}

// ServeMCP runs the MCP server over a stream, as for the stdio transport: one JSON-RPC message on each line in, and one answer on each line out. It returns at the end of the input.
func (s *Server) ServeMCP(in io.Reader, out io.Writer) error {
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 64*1024), 16<<20)
	enc := json.NewEncoder(out)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		if res := s.handle(line); res != nil {
			if err := enc.Encode(res); err != nil {
				return err
			}
		}
	}
	return sc.Err()
}

// serveMCP answers one JSON-RPC message that comes in a POST to /mcp, as the streamable HTTP transport does with a JSON answer.
func (s *Server) serveMCP(w http.ResponseWriter, r *http.Request) {
	msg, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res := s.handle(msg)
	if res == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(res)
}
