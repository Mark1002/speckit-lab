package main

import (
	"bytes"
	"cmp"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"maps"
	"mime"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

//go:embed web
var webFS embed.FS

// The largest valid body is about 20 KB (5,000 four-byte characters), so this leaves headroom.
const maxBody = 64 << 10

type server struct {
	store *store
	log   *slog.Logger
}

func newHandler(s *store, port int, log *slog.Logger) http.Handler {
	a := &server{store: s, log: log}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { a.fail(w, http.StatusNotFound, "", "not found") })
	mux.HandleFunc("GET /api/users", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, http.StatusOK, users) })
	mux.HandleFunc("GET /api/projects", a.requireUser(a.listProjects))
	mux.HandleFunc("POST /api/projects", a.requireUser(a.createProject))
	mux.HandleFunc("GET /api/projects/{pid}", a.requireUser(a.getBoard))
	mux.HandleFunc("POST /api/projects/{pid}/tasks", a.requireUser(a.createTask))
	mux.HandleFunc("GET /api/projects/{pid}/tasks/{tid}", a.requireUser(a.getTask))
	mux.HandleFunc("PATCH /api/projects/{pid}/tasks/{tid}", a.requireUser(a.patchTask))
	mux.HandleFunc("POST /api/projects/{pid}/tasks/{tid}/comments", a.requireUser(a.createComment))
	mux.HandleFunc("PATCH /api/projects/{pid}/tasks/{tid}/comments/{cid}", a.requireUser(a.patchComment))
	mux.HandleFunc("DELETE /api/projects/{pid}/tasks/{tid}/comments/{cid}", a.requireUser(a.deleteComment))

	// Exact paths only: "GET /" would conflict with "/api/" and panic at start-up.
	web, _ := fs.Sub(webFS, "web")
	files := http.FileServerFS(web)
	for _, p := range []string{"GET /{$}", "GET /app.js", "GET /style.css"} {
		mux.Handle(p, files)
	}

	cop := http.NewCrossOriginProtection()
	cop.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.fail(w, http.StatusForbidden, "", "cross-site requests are not allowed")
	}))
	return securityHeaders(a.hostCheck(port, cop.Handler(mux)))
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// hostCheck blocks DNS rebinding: a site that re-points its own domain at 127.0.0.1 still
// sends that domain as Host, so only our own two names get through (research R6).
func (a *server) hostCheck(port int, next http.Handler) http.Handler {
	allowed := []string{"127.0.0.1:" + strconv.Itoa(port), "localhost:" + strconv.Itoa(port)}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !slices.Contains(allowed, r.Host) {
			a.fail(w, http.StatusMisdirectedRequest, "", "requests must use 127.0.0.1 or localhost")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError's field and message are always fixed strings, never request data.
func writeError(w http.ResponseWriter, status int, field, message string) {
	type detail struct {
		Field   string `json:"field,omitempty"`
		Message string `json:"message"`
	}
	writeJSON(w, status, map[string]detail{"error": {field, message}})
}

func (a *server) fail(w http.ResponseWriter, status int, field, message string) {
	a.log.Warn("refused request", "status", status, "field", field, "rule", message)
	writeError(w, status, field, message)
}

func (a *server) serverError(w http.ResponseWriter, err error) {
	a.log.Error("request failed", "err", err)
	writeError(w, http.StatusInternalServerError, "", "something went wrong; nothing was saved")
}

// requireUser trusts X-Taskify-User as a claim, not proof: phase 1 has no login, and
// FR-006 keeps anyone else off this computer.
func (a *server) requireUser(next func(http.ResponseWriter, *http.Request, int)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := r.Header.Get("X-Taskify-User")
		n, err := strconv.Atoi(s)
		if err != nil || strconv.Itoa(n) != s || !validUser(n) {
			a.fail(w, http.StatusUnauthorized, "", "pick a user first")
			return
		}
		next(w, r, n)
	}
}

func pathID(r *http.Request, name string) (int, bool) {
	s := r.PathValue(name)
	n, err := strconv.Atoi(s)
	return n, err == nil && n > 0 && strconv.Itoa(n) == s
}

func (a *server) decodeBody(w http.ResponseWriter, r *http.Request, allowed []string, dst any) bool {
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		a.fail(w, http.StatusUnsupportedMediaType, "", "request body must be JSON")
		return false
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
	if tooBig := (*http.MaxBytesError)(nil); errors.As(err, &tooBig) {
		a.fail(w, http.StatusRequestEntityTooLarge, "", "request body is too large")
		return false
	}
	var fields map[string]json.RawMessage
	dec := json.NewDecoder(bytes.NewReader(b))
	if err == nil {
		err = dec.Decode(&fields)
	}
	if err == nil && dec.Decode(new(json.RawMessage)) != io.EOF {
		err = errors.New("more than one JSON value")
	}
	if err != nil || fields == nil {
		a.fail(w, http.StatusBadRequest, "", "request body must be one JSON object")
		return false
	}
	keys := slices.Sorted(maps.Keys(fields))
	for _, k := range keys {
		if !slices.Contains(allowed, k) {
			a.fail(w, http.StatusBadRequest, "", "request body has a field that is not allowed or has the wrong type")
			return false
		}
	}
	for _, k := range keys {
		if string(bytes.TrimSpace(fields[k])) == "null" {
			a.fail(w, http.StatusBadRequest, k, k+" has the wrong type or is null")
			return false
		}
	}
	if err := json.Unmarshal(b, dst); err != nil {
		var te *json.UnmarshalTypeError
		if errors.As(err, &te) && slices.Contains(allowed, te.Field) {
			a.fail(w, http.StatusBadRequest, te.Field, te.Field+" has the wrong type or is null")
		} else {
			a.fail(w, http.StatusBadRequest, "", "request body has a field that is not allowed or has the wrong type")
		}
		return false
	}
	return true
}

func (a *server) listProjects(w http.ResponseWriter, r *http.Request, user int) {
	var out []Project
	a.store.view(func(d *data) { out = append([]Project{}, d.Projects...) })
	writeJSON(w, http.StatusOK, out)
}

func (a *server) getBoard(w http.ResponseWriter, r *http.Request, user int) {
	var board struct {
		Project
		Tasks []Task `json:"tasks"`
	}
	found := false
	if pid, ok := pathID(r, "pid"); ok {
		a.store.view(func(d *data) {
			p := d.project(pid)
			if p == nil {
				return
			}
			found = true
			board.Project = *p
			board.Tasks = []Task{}
			for _, t := range d.Tasks {
				if t.ProjectID == pid {
					board.Tasks = append(board.Tasks, t)
				}
			}
		})
	}
	if !found {
		a.fail(w, http.StatusNotFound, "", "not found")
		return
	}
	writeJSON(w, http.StatusOK, board)
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

var errNotAuthor = errors.New("not the comment's author")

func (a *server) finish(w http.ResponseWriter, err error, status int, out any) {
	var fe *fieldError
	switch {
	case err == nil && status == http.StatusNoContent:
		w.WriteHeader(status)
	case err == nil:
		writeJSON(w, status, out)
	case errors.Is(err, errNotFound):
		a.fail(w, http.StatusNotFound, "", "not found")
	case errors.Is(err, errNotAuthor):
		a.fail(w, http.StatusForbidden, "", "only the comment's author can change it")
	case errors.As(err, &fe):
		a.fail(w, http.StatusBadRequest, fe.Field, fe.Message)
	default:
		a.serverError(w, err)
	}
}

func (a *server) createTask(w http.ResponseWriter, r *http.Request, user int) {
	var in struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		AssigneeID  int    `json:"assigneeId"`
	}
	if !a.decodeBody(w, r, []string{"title", "description", "assigneeId"}, &in) {
		return
	}
	pid, ok := pathID(r, "pid")
	if !ok {
		a.fail(w, http.StatusNotFound, "", "not found")
		return
	}
	var out Task
	err := a.store.update(func(d *data) error {
		if d.project(pid) == nil {
			return errNotFound
		}
		if fe := cmp.Or(
			checkText("title", in.Title, true, 200, false),
			checkText("description", in.Description, false, 5000, true),
			checkAssignee(in.AssigneeID),
		); fe != nil {
			return fe
		}
		out = Task{
			ID: d.newID(), ProjectID: pid, Title: in.Title, Description: in.Description,
			Status: "todo", AssigneeID: in.AssigneeID, CreatedBy: user, CreatedAt: now(),
		}
		d.Tasks = append(d.Tasks, out)
		return nil
	})
	a.finish(w, err, http.StatusCreated, out)
}

func (a *server) getTask(w http.ResponseWriter, r *http.Request, user int) {
	var detail struct {
		Task
		Comments []Comment `json:"comments"`
	}
	found := false
	pid, okP := pathID(r, "pid")
	tid, okT := pathID(r, "tid")
	if okP && okT {
		a.store.view(func(d *data) {
			t := d.task(pid, tid)
			if t == nil {
				return
			}
			found = true
			detail.Task = *t
			detail.Comments = []Comment{}
			for _, c := range d.Comments {
				if c.TaskID == tid {
					detail.Comments = append(detail.Comments, c)
				}
			}
		})
	}
	if !found {
		a.fail(w, http.StatusNotFound, "", "not found")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (a *server) patchTask(w http.ResponseWriter, r *http.Request, user int) {
	var in struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Status      *string `json:"status"`
		AssigneeID  *int    `json:"assigneeId"`
	}
	if !a.decodeBody(w, r, []string{"title", "description", "status", "assigneeId"}, &in) {
		return
	}
	if in.Title == nil && in.Description == nil && in.Status == nil && in.AssigneeID == nil {
		a.fail(w, http.StatusBadRequest, "", "request body must change at least one field")
		return
	}
	pid, okP := pathID(r, "pid")
	tid, okT := pathID(r, "tid")
	if !okP || !okT {
		a.fail(w, http.StatusNotFound, "", "not found")
		return
	}
	var out Task
	err := a.store.update(func(d *data) error {
		t := d.task(pid, tid)
		if t == nil {
			return errNotFound
		}
		next := *t
		if in.Title != nil {
			next.Title = *in.Title
		}
		if in.Description != nil {
			next.Description = *in.Description
		}
		if in.Status != nil {
			next.Status = *in.Status
		}
		if in.AssigneeID != nil {
			next.AssigneeID = *in.AssigneeID
		}
		if fe := cmp.Or(
			checkText("title", next.Title, true, 200, false),
			checkText("description", next.Description, false, 5000, true),
			checkStatus(next.Status),
			checkAssignee(next.AssigneeID),
		); fe != nil {
			return fe
		}
		*t, out = next, next
		return nil
	})
	a.finish(w, err, http.StatusOK, out)
}

func (a *server) createComment(w http.ResponseWriter, r *http.Request, user int) {
	var in struct {
		Text string `json:"text"`
	}
	if !a.decodeBody(w, r, []string{"text"}, &in) {
		return
	}
	pid, okP := pathID(r, "pid")
	tid, okT := pathID(r, "tid")
	if !okP || !okT {
		a.fail(w, http.StatusNotFound, "", "not found")
		return
	}
	var out Comment
	err := a.store.update(func(d *data) error {
		if d.task(pid, tid) == nil {
			return errNotFound
		}
		if fe := checkText("text", in.Text, true, 2000, true); fe != nil {
			return fe
		}
		out = Comment{ID: d.newID(), TaskID: tid, AuthorID: user, Text: in.Text, CreatedAt: now()}
		d.Comments = append(d.Comments, out)
		return nil
	})
	a.finish(w, err, http.StatusCreated, out)
}

func commentIDs(r *http.Request) (pid, tid, cid int, ok bool) {
	pid, okP := pathID(r, "pid")
	tid, okT := pathID(r, "tid")
	cid, okC := pathID(r, "cid")
	return pid, tid, cid, okP && okT && okC
}

func (a *server) patchComment(w http.ResponseWriter, r *http.Request, user int) {
	var in struct {
		Text *string `json:"text"`
	}
	if !a.decodeBody(w, r, []string{"text"}, &in) {
		return
	}
	if in.Text == nil {
		a.fail(w, http.StatusBadRequest, "", "request body must change at least one field")
		return
	}
	pid, tid, cid, ok := commentIDs(r)
	if !ok {
		a.fail(w, http.StatusNotFound, "", "not found")
		return
	}
	var out Comment
	err := a.store.update(func(d *data) error {
		i := d.comment(pid, tid, cid)
		switch {
		case i < 0:
			return errNotFound
		case d.Comments[i].AuthorID != user:
			return errNotAuthor
		}
		if fe := checkText("text", *in.Text, true, 2000, true); fe != nil {
			return fe
		}
		d.Comments[i].Text = *in.Text
		out = d.Comments[i]
		return nil
	})
	a.finish(w, err, http.StatusOK, out)
}

func (a *server) deleteComment(w http.ResponseWriter, r *http.Request, user int) {
	pid, tid, cid, ok := commentIDs(r)
	if !ok {
		a.fail(w, http.StatusNotFound, "", "not found")
		return
	}
	err := a.store.update(func(d *data) error {
		i := d.comment(pid, tid, cid)
		switch {
		case i < 0:
			return errNotFound
		case d.Comments[i].AuthorID != user:
			return errNotAuthor
		}
		d.Comments = slices.Delete(d.Comments, i, i+1)
		return nil
	})
	a.finish(w, err, http.StatusNoContent, nil)
}

func (a *server) createProject(w http.ResponseWriter, r *http.Request, user int) {
	var in struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if !a.decodeBody(w, r, []string{"name", "description"}, &in) {
		return
	}
	var out Project
	// The duplicate check runs inside update so no other request can add the same name in between.
	err := a.store.update(func(d *data) error {
		if fe := cmp.Or(
			checkText("name", in.Name, true, 100, false),
			checkText("description", in.Description, false, 1000, true),
		); fe != nil {
			return fe
		}
		for _, p := range d.Projects {
			if strings.EqualFold(p.Name, in.Name) {
				return &fieldError{"name", "name is already used by another project"}
			}
		}
		out = Project{ID: d.newID(), Name: in.Name, Description: in.Description, CreatedBy: user, CreatedAt: now()}
		d.Projects = append(d.Projects, out)
		return nil
	})
	a.finish(w, err, http.StatusCreated, out)
}
