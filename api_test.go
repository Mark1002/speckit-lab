package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

func newTestServer(t *testing.T) (http.Handler, *store) {
	t.Helper()
	s, err := openStore(filepath.Join(t.TempDir(), "taskify.json"), seedTime)
	if err != nil {
		t.Fatal(err)
	}
	return newHandler(s, 8080, discard), s
}

// newReq builds a same-origin request as the Taskify page would send it.
func newReq(method, path, user, body string) *http.Request {
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	r := httptest.NewRequest(method, path, rd)
	r.Host = "127.0.0.1:8080"
	if user != "" {
		r.Header.Set("X-Taskify-User", user)
	}
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}

func serve(h http.Handler, r *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

type apiError struct {
	Error struct {
		Field   *string `json:"field"`
		Message string  `json:"message"`
	} `json:"error"`
}

func errorOf(t *testing.T, rec *httptest.ResponseRecorder) apiError {
	t.Helper()
	var e apiError
	if err := json.Unmarshal(rec.Body.Bytes(), &e); err != nil {
		t.Fatalf("error body %q is not JSON: %v", rec.Body.String(), err)
	}
	return e
}

func wantError(t *testing.T, rec *httptest.ResponseRecorder, status int, field, message string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, status, rec.Body)
	}
	e := errorOf(t, rec)
	if e.Error.Message != message {
		t.Errorf("message = %q, want %q", e.Error.Message, message)
	}
	switch {
	case field == "" && e.Error.Field != nil:
		t.Errorf("field = %q, want none", *e.Error.Field)
	case field != "" && (e.Error.Field == nil || *e.Error.Field != field):
		t.Errorf("field = %v, want %q", e.Error.Field, field)
	}
}

func TestHostMustBeLoopbackWithPort(t *testing.T) {
	h, _ := newTestServer(t)
	for _, host := range []string{"127.0.0.1:8080", "localhost:8080"} {
		r := newReq("GET", "/api/users", "", "")
		r.Host = host
		if rec := serve(h, r); rec.Code != http.StatusOK {
			t.Errorf("Host %s: status %d, want 200", host, rec.Code)
		}
	}
	for _, host := range []string{"evil.example:8080", "127.0.0.1:9999", "localhost"} {
		for _, path := range []string{"/api/users", "/"} {
			r := newReq("GET", path, "", "")
			r.Host = host
			wantError(t, serve(h, r), http.StatusMisdirectedRequest, "", "requests must use 127.0.0.1 or localhost")
		}
	}
}

func TestCrossSiteWriteIsRefused(t *testing.T) {
	h, _ := newTestServer(t)
	r := newReq("POST", "/api/projects", "1", `{"name":"x"}`)
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	wantError(t, serve(h, r), http.StatusForbidden, "", "cross-site requests are not allowed")
}

func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	h, _ := newTestServer(t)
	badHost := newReq("GET", "/api/users", "", "")
	badHost.Host = "evil.example:8080"
	for _, r := range []*http.Request{
		newReq("GET", "/api/users", "", ""),
		newReq("GET", "/api/nope", "1", ""),
		badHost,
	} {
		rec := serve(h, r)
		hd := rec.Header()
		if got := hd.Get("Content-Security-Policy"); got != "default-src 'self'; frame-ancestors 'none'" {
			t.Errorf("%d: CSP = %q", rec.Code, got)
		}
		if got := hd.Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("%d: X-Content-Type-Options = %q", rec.Code, got)
		}
		if got := hd.Get("Referrer-Policy"); got != "no-referrer" {
			t.Errorf("%d: Referrer-Policy = %q", rec.Code, got)
		}
		if got := hd.Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("%d: unexpected Access-Control-Allow-Origin %q", rec.Code, got)
		}
	}
}

func TestUsersAreListedBeforeAUserIsPicked(t *testing.T) {
	h, _ := newTestServer(t)
	rec := serve(h, newReq("GET", "/api/users", "", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	var got []User
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	want := []User{
		{1, "Avery Chen", "product_manager"},
		{2, "Blake Rivera", "engineer"},
		{3, "Casey Patel", "engineer"},
		{4, "Devon Kim", "engineer"},
		{5, "Emery Novak", "engineer"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("users = %+v, want %+v", got, want)
	}
}

func TestProjectsNeedAValidPickedUser(t *testing.T) {
	h, _ := newTestServer(t)
	for _, user := range []string{"", "0", "6", "abc", "1 ", "+1", "01"} {
		wantError(t, serve(h, newReq("GET", "/api/projects", user, "")), http.StatusUnauthorized, "", "pick a user first")
	}

	rec := serve(h, newReq("GET", "/api/projects", "2", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	var got []Project
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	var ids []int
	var names []string
	for _, p := range got {
		ids = append(ids, p.ID)
		names = append(names, p.Name)
	}
	if !reflect.DeepEqual(ids, []int{1, 2, 3}) || !reflect.DeepEqual(names, []string{"Website Refresh", "Mobile App Beta", "Internal Tools"}) {
		t.Fatalf("projects = %v %q", ids, names)
	}
}

func TestUnknownAPIPathIsJSON404(t *testing.T) {
	h, _ := newTestServer(t)
	wantError(t, serve(h, newReq("GET", "/api/nope", "1", "")), http.StatusNotFound, "", "not found")
}

func TestDecodeBody(t *testing.T) {
	a := &server{log: discard}
	type titleOnly struct {
		Title *string `json:"title"`
	}
	run := func(contentType, body string) (*httptest.ResponseRecorder, bool, titleOnly) {
		r := httptest.NewRequest("POST", "/api/x", strings.NewReader(body))
		if contentType != "" {
			r.Header.Set("Content-Type", contentType)
		}
		rec := httptest.NewRecorder()
		var dst titleOnly
		ok := a.decodeBody(rec, r, []string{"title"}, &dst)
		return rec, ok, dst
	}

	rec, ok, dst := run("application/json; charset=utf-8", `{"title":"ok"}`)
	if !ok || dst.Title == nil || *dst.Title != "ok" {
		t.Fatalf("valid body refused: ok=%v status=%d", ok, rec.Code)
	}

	refused := []struct {
		name, contentType, body string
		status                  int
		field, message          string
	}{
		{"no content type", "", `{"title":"x"}`, 415, "", "request body must be JSON"},
		{"text content type", "text/plain", `{"title":"x"}`, 415, "", "request body must be JSON"},
		{"too large", "application/json", strings.Repeat(" ", 64<<10+1), 413, "", "request body is too large"},
		{"array", "application/json", `[]`, 400, "", "request body must be one JSON object"},
		{"two values", "application/json", `{"a":1}{"b":2}`, 400, "", "request body must be one JSON object"},
		{"null body", "application/json", `null`, 400, "", "request body must be one JSON object"},
		{"unlisted field", "application/json", `{"evil_field":1}`, 400, "", "request body has a field that is not allowed or has the wrong type"},
		{"null listed field", "application/json", `{"title":null}`, 400, "title", "title has the wrong type or is null"},
		{"wrong type listed field", "application/json", `{"title":5}`, 400, "title", "title has the wrong type or is null"},
	}
	for _, tc := range refused {
		t.Run(tc.name, func(t *testing.T) {
			rec, ok, _ := run(tc.contentType, tc.body)
			if ok {
				t.Fatal("want refused, got accepted")
			}
			wantError(t, rec, tc.status, tc.field, tc.message)
			if strings.Contains(rec.Body.String(), "evil_field") {
				t.Fatal("error repeats an unlisted field name")
			}
		})
	}
}

func TestStaticFiles(t *testing.T) {
	h, _ := newTestServer(t)
	rec := serve(h, newReq("GET", "/", "", ""))
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("GET /: status %d, type %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	for _, path := range []string{"/app.js", "/style.css"} {
		if rec := serve(h, newReq("GET", path, "", "")); rec.Code != http.StatusOK {
			t.Errorf("GET %s: status %d, want 200", path, rec.Code)
		}
	}
	for _, path := range []string{"/index.html", "/x.js"} {
		if rec := serve(h, newReq("GET", path, "", "")); rec.Code != http.StatusNotFound {
			t.Errorf("GET %s: status %d, want 404", path, rec.Code)
		}
	}
}

func TestListenAddrIsLoopbackOnly(t *testing.T) {
	if got := listenAddr(8080); got != "127.0.0.1:8080" {
		t.Fatalf("listenAddr(8080) = %q, want 127.0.0.1:8080", got)
	}
}

// Guards SC-007: user text must never reach an HTML-parsing sink.
func TestAppJSHasNoHTMLSinks(t *testing.T) {
	src, err := webFS.ReadFile("web/app.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, sink := range []string{"innerHTML", "outerHTML", "insertAdjacentHTML", "document.write"} {
		if strings.Contains(string(src), sink) {
			t.Errorf("web/app.js uses %s", sink)
		}
	}
}

type boardResp struct {
	Project
	Tasks []Task `json:"tasks"`
}

func getBoard(t *testing.T, h http.Handler, pid string) boardResp {
	t.Helper()
	rec := serve(h, newReq("GET", "/api/projects/"+pid, "1", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET board %s: status %d (body %s)", pid, rec.Code, rec.Body)
	}
	var b boardResp
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	return b
}

func taskIn(b boardResp, id int) Task {
	for _, task := range b.Tasks {
		if task.ID == id {
			return task
		}
	}
	return Task{}
}

func TestBoardListsOnlyThatProjectsTasksInIDOrder(t *testing.T) {
	h, _ := newTestServer(t)
	b := getBoard(t, h, "1")
	if b.ID != 1 || b.Name != "Website Refresh" || b.CreatedBy != 1 {
		t.Fatalf("project = %+v", b.Project)
	}
	var ids []int
	for _, task := range b.Tasks {
		ids = append(ids, task.ID)
	}
	if !reflect.DeepEqual(ids, []int{4, 5, 6, 7, 8}) {
		t.Fatalf("task IDs = %v, want [4 5 6 7 8]", ids)
	}
}

func TestBoardRefusesBadProjectIDsAndMissingUser(t *testing.T) {
	h, _ := newTestServer(t)
	for _, pid := range []string{"999", "abc", "0", "-1", "+1", "01"} {
		wantError(t, serve(h, newReq("GET", "/api/projects/"+pid, "1", "")), http.StatusNotFound, "", "not found")
	}
	wantError(t, serve(h, newReq("GET", "/api/projects/1", "", "")), http.StatusUnauthorized, "", "pick a user first")
}

func TestEmptyBoardHasEmptyTaskListNotNull(t *testing.T) {
	h, s := newTestServer(t)
	var pid int
	if err := s.update(func(d *data) error {
		pid = d.newID()
		d.Projects = append(d.Projects, Project{ID: pid, Name: "Empty", CreatedBy: 1, CreatedAt: "2026-10-08T09:00:00Z"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rec := serve(h, newReq("GET", "/api/projects/"+strconv.Itoa(pid), "1", ""))
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("status %d, body %s", rec.Code, rec.Body)
	}
	if string(raw["tasks"]) != "[]" {
		t.Fatalf(`"tasks" = %s, want []`, raw["tasks"])
	}
}

func TestMoveTaskIsSavedAndKeepsAssignee(t *testing.T) {
	h, s := newTestServer(t)
	rec := serve(h, newReq("PATCH", "/api/projects/1/tasks/4", "1", `{"status":"in_review"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d (body %s)", rec.Code, rec.Body)
	}
	var got Task
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != 4 || got.Status != "in_review" || got.AssigneeID != 1 {
		t.Fatalf("response task = %+v", got)
	}

	again, err := openStore(s.path, seedTime)
	if err != nil {
		t.Fatal(err)
	}
	if st := snapshot(again).Tasks[0].Status; st != "in_review" {
		t.Fatalf("after reopen: status %q, want in_review", st)
	}

	if rec := serve(h, newReq("PATCH", "/api/projects/1/tasks/4", "1", `{"status":"in_review"}`)); rec.Code != http.StatusOK {
		t.Fatalf("same status again: status %d, want 200", rec.Code)
	}
	if st := taskIn(getBoard(t, h, "1"), 4).Status; st != "in_review" {
		t.Fatalf("same status again changed it to %q", st)
	}
}

func TestAnyUserMovesAnyTaskAndAssigneeNeverChanges(t *testing.T) {
	h, _ := newTestServer(t)
	moves := []struct {
		user, tid, status string
		id, assignee      int
	}{
		{"5", "6", "done", 6, 2},        // Q1: user 5 moves a task assigned to user 2
		{"1", "5", "done", 5, 0},        // Q4: unassigned task to done stays unassigned
		{"2", "7", "todo", 7, 3},        // an assigned task keeps its assignee
		{"3", "5", "in_progress", 5, 0}, // and moves back out of done
	}
	for _, m := range moves {
		rec := serve(h, newReq("PATCH", "/api/projects/1/tasks/"+m.tid, m.user, `{"status":"`+m.status+`"}`))
		if rec.Code != http.StatusOK {
			t.Fatalf("user %s moving task %s: status %d (body %s)", m.user, m.tid, rec.Code, rec.Body)
		}
		task := taskIn(getBoard(t, h, "1"), m.id)
		if task.Status != m.status || task.AssigneeID != m.assignee {
			t.Fatalf("task %d = status %q assignee %d, want %q %d", m.id, task.Status, task.AssigneeID, m.status, m.assignee)
		}
	}
}

func TestMoveTaskRefusalsSaveNothing(t *testing.T) {
	h, s := newTestServer(t)
	before, _ := os.ReadFile(s.path)
	refused := []struct {
		name, user, path, body string
		status                 int
		field, message         string
	}{
		{"not a status", "1", "/api/projects/1/tasks/4", `{"status":"Done"}`, 400, "status", "status must be one of todo, in_progress, in_review, done"},
		{"empty body", "1", "/api/projects/1/tasks/4", `{}`, 400, "", "request body must change at least one field"},
		{"null status", "1", "/api/projects/1/tasks/4", `{"status":null}`, 400, "status", "status has the wrong type or is null"},
		{"system field", "1", "/api/projects/1/tasks/4", `{"createdBy":1}`, 400, "", "request body has a field that is not allowed or has the wrong type"},
		{"task in another project", "1", "/api/projects/1/tasks/9", `{"status":"done"}`, 404, "", "not found"},
		{"no such task", "1", "/api/projects/1/tasks/999", `{"status":"done"}`, 404, "", "not found"},
		{"padded task id", "1", "/api/projects/1/tasks/04", `{"status":"done"}`, 404, "", "not found"},
		{"no user", "", "/api/projects/1/tasks/4", `{"status":"done"}`, 401, "", "pick a user first"},
	}
	for _, tc := range refused {
		t.Run(tc.name, func(t *testing.T) {
			wantError(t, serve(h, newReq("PATCH", tc.path, tc.user, tc.body)), tc.status, tc.field, tc.message)
			if after, _ := os.ReadFile(s.path); string(after) != string(before) {
				t.Fatal("data file changed after a refused request")
			}
		})
	}
}

func TestMoveTaskSaveFailureReturns500AndKeepsOldStatus(t *testing.T) {
	h, s := newTestServer(t)
	if err := os.RemoveAll(filepath.Dir(s.path)); err != nil {
		t.Fatal(err)
	}
	wantError(t, serve(h, newReq("PATCH", "/api/projects/1/tasks/4", "1", `{"status":"done"}`)),
		http.StatusInternalServerError, "", "something went wrong; nothing was saved")
	if st := taskIn(getBoard(t, h, "1"), 4).Status; st != "todo" {
		t.Fatalf("status after failed save = %q, want todo", st)
	}
}
