package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
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

func TestRoutesWithoutAPickedUserAreRefused(t *testing.T) {
	h, s := newTestServer(t)
	c := postComment(t, h, "1", "Mine")
	comment := commentsPath + "/" + strconv.Itoa(c.ID)
	for _, r := range []struct{ method, path, body string }{
		{"POST", "/api/projects", `{"name":"x"}`},
		{"POST", "/api/projects/1/tasks", `{"title":"x"}`},
		{"GET", "/api/projects/1/tasks/4", ""},
		{"POST", commentsPath, `{"text":"x"}`},
		{"PATCH", comment, `{"text":"x"}`},
		{"DELETE", comment, ""},
	} {
		t.Run(r.method+" "+r.path, func(t *testing.T) {
			before, _ := os.ReadFile(s.path)
			wantError(t, serve(h, newReq(r.method, r.path, "", r.body)), http.StatusUnauthorized, "", "pick a user first")
			if after, _ := os.ReadFile(s.path); string(after) != string(before) {
				t.Fatal("data file changed after a refused request")
			}
		})
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

func jsonBody(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func decodeInto[T any](t *testing.T, rec *httptest.ResponseRecorder, status int) T {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, status, rec.Body)
	}
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

type taskDetail struct {
	Task
	Comments []Comment `json:"comments"`
}

func getTask(t *testing.T, h http.Handler, path string) taskDetail {
	t.Helper()
	return decodeInto[taskDetail](t, serve(h, newReq("GET", path, "1", "")), http.StatusOK)
}

func TestCreateTaskDefaultsAndStoresInputExactly(t *testing.T) {
	h, _ := newTestServer(t)
	got := decodeInto[Task](t, serve(h, newReq("POST", "/api/projects/1/tasks", "3", `{"title":"Fix header"}`)), http.StatusCreated)
	if got.ProjectID != 1 || got.Title != "Fix header" || got.Status != "todo" || got.AssigneeID != 0 || got.Description != "" || got.CreatedBy != 3 {
		t.Fatalf("created task = %+v", got)
	}
	if at, err := time.Parse(time.RFC3339, got.CreatedAt); err != nil || at.Location() != time.UTC {
		t.Fatalf("createdAt %q is not RFC 3339 UTC", got.CreatedAt)
	}
	if got.ID <= 17 {
		t.Fatalf("new task ID %d is not above the sample IDs (max 17)", got.ID)
	}

	second := decodeInto[Task](t, serve(h, newReq("POST", "/api/projects/1/tasks", "3",
		jsonBody(map[string]any{"title": "Two", "description": "a\nb", "assigneeId": 3}))), http.StatusCreated)
	stored := getTask(t, h, "/api/projects/1/tasks/"+strconv.Itoa(second.ID))
	if stored.Description != "a\nb" || stored.AssigneeID != 3 {
		t.Fatalf("stored task = %+v", stored.Task)
	}
}

func TestCreateTaskAcceptsLimits(t *testing.T) {
	h, _ := newTestServer(t)
	for _, body := range []map[string]any{
		{"title": e(200)},
		{"title": "Long description", "description": e(5000)},
	} {
		if rec := serve(h, newReq("POST", "/api/projects/1/tasks", "1", jsonBody(body))); rec.Code != http.StatusCreated {
			t.Fatalf("status %d (body %s)", rec.Code, rec.Body)
		}
	}
}

func TestCreateTaskRefusalsSaveNothing(t *testing.T) {
	h, s := newTestServer(t)
	refused := []struct {
		name, path, contentType, body string
		status                        int
		field, message                string
	}{
		{"title over limit", "/api/projects/1/tasks", "", jsonBody(map[string]any{"title": e(201)}), 400, "title", "title must be 1 to 200 characters"},
		{"title spaces", "/api/projects/1/tasks", "", `{"title":"   "}`, 400, "title", "title must be 1 to 200 characters"},
		{"title line break", "/api/projects/1/tasks", "", jsonBody(map[string]any{"title": "a\nb"}), 400, "title", "title must be a single line"},
		{"description over limit", "/api/projects/1/tasks", "", jsonBody(map[string]any{"title": "x", "description": e(5001)}), 400, "description", "description must be at most 5000 characters"},
		{"description CR", "/api/projects/1/tasks", "", jsonBody(map[string]any{"title": "x", "description": "a\rb"}), 400, "description", "description must not contain control characters"},
		{"assignee not a user", "/api/projects/1/tasks", "", `{"title":"x","assigneeId":6}`, 400, "assigneeId", "assigneeId must be 0 or a user ID from 1 to 5"},
		{"assignee as string", "/api/projects/1/tasks", "", `{"title":"x","assigneeId":"3"}`, 400, "assigneeId", "assigneeId has the wrong type or is null"},
		{"status on create", "/api/projects/1/tasks", "", `{"title":"x","status":"done"}`, 400, "", "request body has a field that is not allowed or has the wrong type"},
		{"system field", "/api/projects/1/tasks", "", `{"title":"x","projectId":2}`, 400, "", "request body has a field that is not allowed or has the wrong type"},
		{"null title", "/api/projects/1/tasks", "", `{"title":null}`, 400, "title", "title has the wrong type or is null"},
		{"invalid UTF-8", "/api/projects/1/tasks", "", "{\"title\":\"bad\xffbyte\"}", 400, "", "request body must be one JSON object"},
		{"lone high surrogate", "/api/projects/1/tasks", "", `{"title":"bad\ud800esc"}`, 400, "", "request body must be one JSON object"},
		{"lone low surrogate", "/api/projects/1/tasks", "", `{"title":"bad\udc00esc"}`, 400, "", "request body must be one JSON object"},
		{"high surrogate without its pair", "/api/projects/1/tasks", "", `{"title":"bad\ud800\u0041"}`, 400, "", "request body must be one JSON object"},
		{"too large", "/api/projects/1/tasks", "", jsonBody(map[string]any{"title": strings.Repeat("a", 64<<10)}), 413, "", "request body is too large"},
		{"not JSON", "/api/projects/1/tasks", "text/plain", `{"title":"x"}`, 415, "", "request body must be JSON"},
		{"unknown project", "/api/projects/999/tasks", "", `{"title":"x"}`, 404, "", "not found"},
	}
	for _, tc := range refused {
		t.Run(tc.name, func(t *testing.T) {
			r := newReq("POST", tc.path, "1", tc.body)
			if tc.contentType != "" {
				r.Header.Set("Content-Type", tc.contentType)
			}
			wantError(t, serve(h, r), tc.status, tc.field, tc.message)
			if n := len(snapshot(s).Tasks); n != 14 {
				t.Fatalf("task count = %d after a refused create, want 14", n)
			}
		})
	}
}

func TestTaskTextIsStoredAsPlainText(t *testing.T) {
	h, _ := newTestServer(t)
	const title = "<script>alert(1)</script>"
	created := decodeInto[Task](t, serve(h, newReq("POST", "/api/projects/1/tasks", "1", jsonBody(map[string]any{"title": title}))), http.StatusCreated)
	if got := getTask(t, h, "/api/projects/1/tasks/"+strconv.Itoa(created.ID)).Title; got != title {
		t.Fatalf("title read back as %q, want %q", got, title)
	}
}

func TestValidUnicodeIsStoredExactly(t *testing.T) {
	h, _ := newTestServer(t)
	for body, want := range map[string]string{
		"{\"title\":\"typed \uFFFD\"}":  "typed \uFFFD",
		`{"title":"escaped \ufffd"}`:    "escaped \uFFFD",
		`{"title":"pair \ud83d\ude00"}`: "pair 😀",
		`{"title":"text \\ud800"}`:      `text \ud800`,
	} {
		created := decodeInto[Task](t, serve(h, newReq("POST", "/api/projects/1/tasks", "1", body)), http.StatusCreated)
		if got := getTask(t, h, "/api/projects/1/tasks/"+strconv.Itoa(created.ID)).Title; got != want {
			t.Errorf("title read back as %q, want %q", got, want)
		}
	}
}

func TestGetTaskHasEmptyCommentListAndChecksNesting(t *testing.T) {
	h, _ := newTestServer(t)
	rec := serve(h, newReq("GET", "/api/projects/1/tasks/4", "1", ""))
	detail := decodeInto[taskDetail](t, rec, http.StatusOK)
	if detail.ID != 4 || detail.Title != "Draft new homepage copy" || detail.CreatedBy != 1 {
		t.Fatalf("task = %+v", detail.Task)
	}
	var raw map[string]json.RawMessage
	json.Unmarshal(rec.Body.Bytes(), &raw)
	if string(raw["comments"]) != "[]" {
		t.Fatalf(`"comments" = %s, want []`, raw["comments"])
	}
	wantError(t, serve(h, newReq("GET", "/api/projects/1/tasks/9", "1", "")), http.StatusNotFound, "", "not found")
}

func TestAnyUserAssignsAndEditsTasks(t *testing.T) {
	h, _ := newTestServer(t)
	patch := func(user, body string) Task {
		t.Helper()
		return decodeInto[Task](t, serve(h, newReq("PATCH", "/api/projects/1/tasks/4", user, body)), http.StatusOK)
	}
	if got := patch("2", `{"assigneeId":4}`); got.AssigneeID != 4 || got.Status != "todo" {
		t.Fatalf("after assign (Q2): %+v", got)
	}
	if got := patch("5", `{"assigneeId":0}`); got.AssigneeID != 0 {
		t.Fatalf("after clearing assignee: %+v", got)
	}
	if got := patch("3", `{"title":"New","description":""}`); got.Title != "New" || got.Description != "" || got.Status != "todo" {
		t.Fatalf("after edit: %+v", got)
	}

	wantError(t, serve(h, newReq("PATCH", "/api/projects/1/tasks/4", "1", `{"assigneeId":null}`)), 400, "assigneeId", "assigneeId has the wrong type or is null")
	wantError(t, serve(h, newReq("PATCH", "/api/projects/1/tasks/4", "1", jsonBody(map[string]any{"title": "a\nb"}))), 400, "title", "title must be a single line")
	if got := getTask(t, h, "/api/projects/1/tasks/4"); got.Title != "New" {
		t.Fatalf("refused edit changed the title to %q", got.Title)
	}
}

const commentsPath = "/api/projects/1/tasks/4/comments"

func postComment(t *testing.T, h http.Handler, user, text string) Comment {
	t.Helper()
	return decodeInto[Comment](t, serve(h, newReq("POST", commentsPath, user, jsonBody(map[string]any{"text": text}))), http.StatusCreated)
}

func TestCreateCommentsListedInIDOrder(t *testing.T) {
	h, _ := newTestServer(t)
	first := postComment(t, h, "3", "Looks good")
	if first.AuthorID != 3 || first.TaskID != 4 || first.Text != "Looks good" {
		t.Fatalf("comment = %+v", first)
	}
	if _, err := time.Parse(time.RFC3339, first.CreatedAt); err != nil {
		t.Fatalf("createdAt %q is not RFC 3339", first.CreatedAt)
	}
	second := postComment(t, h, "1", "Agreed")
	var ids []int
	for _, c := range getTask(t, h, "/api/projects/1/tasks/4").Comments {
		ids = append(ids, c.ID)
	}
	if !reflect.DeepEqual(ids, []int{first.ID, second.ID}) || second.ID <= first.ID {
		t.Fatalf("comment IDs = %v, want [%d %d] in increasing order", ids, first.ID, second.ID)
	}
}

func TestCommentTextRules(t *testing.T) {
	h, s := newTestServer(t)
	postComment(t, h, "1", e(2000))
	postComment(t, h, "1", "a\nb")
	refused := []struct {
		name, path, body string
		status           int
		field, message   string
	}{
		{"over limit", commentsPath, jsonBody(map[string]any{"text": e(2001)}), 400, "text", "text must be 1 to 2000 characters"},
		{"only whitespace", commentsPath, jsonBody(map[string]any{"text": "  \n "}), 400, "text", "text must be 1 to 2000 characters"},
		{"carriage return", commentsPath, jsonBody(map[string]any{"text": "a\rb"}), 400, "text", "text must not contain control characters"},
		{"author in body", commentsPath, `{"authorId":2,"text":"x"}`, 400, "", "request body has a field that is not allowed or has the wrong type"},
		{"task in another project", "/api/projects/1/tasks/9/comments", `{"text":"x"}`, 404, "", "not found"},
	}
	for _, tc := range refused {
		t.Run(tc.name, func(t *testing.T) {
			wantError(t, serve(h, newReq("POST", tc.path, "1", tc.body)), tc.status, tc.field, tc.message)
			if n := len(snapshot(s).Comments); n != 2 {
				t.Fatalf("comment count = %d after a refused create, want 2", n)
			}
		})
	}
}

func TestOnlyTheAuthorEditsAComment(t *testing.T) {
	h, s := newTestServer(t)
	c := postComment(t, h, "3", "Looks good")
	path := commentsPath + "/" + strconv.Itoa(c.ID)
	before, _ := os.ReadFile(s.path)

	wantError(t, serve(h, newReq("PATCH", path, "1", `{"text":"Hijacked"}`)), http.StatusForbidden, "", "only the comment's author can change it")
	// Ownership is checked before field rules, so a non-author learns nothing about validation.
	wantError(t, serve(h, newReq("PATCH", path, "1", jsonBody(map[string]any{"text": e(2001)}))), http.StatusForbidden, "", "only the comment's author can change it")
	if after, _ := os.ReadFile(s.path); string(after) != string(before) {
		t.Fatal("data file changed after a refused edit")
	}

	rec := serve(h, newReq("PATCH", path, "3", `{"text":"Edited"}`))
	var raw map[string]json.RawMessage
	edited := decodeInto[Comment](t, rec, http.StatusOK)
	json.Unmarshal(rec.Body.Bytes(), &raw)
	if edited.Text != "Edited" || edited.CreatedAt != c.CreatedAt || edited.AuthorID != 3 {
		t.Fatalf("edited comment = %+v, want text Edited and createdAt %q", edited, c.CreatedAt)
	}
	if len(raw) != 5 {
		t.Fatalf("edited comment has fields %v; FR-022 allows no edited mark", slices.Collect(maps.Keys(raw)))
	}
	if got := getTask(t, h, "/api/projects/1/tasks/4").Comments[0].Text; got != "Edited" {
		t.Fatalf("stored text = %q, want Edited", got)
	}
}

func TestOnlyTheAuthorDeletesAComment(t *testing.T) {
	h, s := newTestServer(t)
	c := postComment(t, h, "3", "Looks good")
	path := commentsPath + "/" + strconv.Itoa(c.ID)

	wantError(t, serve(h, newReq("DELETE", path, "1", "")), http.StatusForbidden, "", "only the comment's author can change it")
	if n := len(getTask(t, h, "/api/projects/1/tasks/4").Comments); n != 1 {
		t.Fatalf("comment count after a refused delete = %d, want 1", n)
	}
	wantError(t, serve(h, newReq("DELETE", "/api/projects/1/tasks/5/comments/"+strconv.Itoa(c.ID), "3", "")), http.StatusNotFound, "", "not found")

	if rec := serve(h, newReq("DELETE", path, "3", "")); rec.Code != http.StatusNoContent || rec.Body.Len() != 0 {
		t.Fatalf("author delete: status %d, body %q; want 204 and no body", rec.Code, rec.Body)
	}
	if n := len(getTask(t, h, "/api/projects/1/tasks/4").Comments); n != 0 {
		t.Fatalf("comment still listed after delete")
	}
	again, err := openStore(s.path, seedTime)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(snapshot(again).Comments); n != 0 {
		t.Fatalf("deleted comment came back after reopening the store")
	}
	wantError(t, serve(h, newReq("DELETE", path, "3", "")), http.StatusNotFound, "", "not found")
	wantError(t, serve(h, newReq("PATCH", path, "3", `{"text":"x"}`)), http.StatusNotFound, "", "not found")

	if next := postComment(t, h, "3", "New"); next.ID <= c.ID {
		t.Fatalf("new comment ID %d reuses or precedes deleted ID %d", next.ID, c.ID)
	}
}

func TestCreateProjectAppearsWithEmptyBoard(t *testing.T) {
	h, _ := newTestServer(t)
	p := decodeInto[Project](t, serve(h, newReq("POST", "/api/projects", "2", `{"name":"Website"}`)), http.StatusCreated)
	if p.Name != "Website" || p.Description != "" || p.CreatedBy != 2 || p.ID <= 17 {
		t.Fatalf("created project = %+v", p)
	}
	list := decodeInto[[]Project](t, serve(h, newReq("GET", "/api/projects", "4", "")), http.StatusOK)
	if len(list) != 4 || list[3].ID != p.ID {
		t.Fatalf("project list = %+v, want the new project last", list)
	}
	rec := serve(h, newReq("GET", "/api/projects/"+strconv.Itoa(p.ID), "4", ""))
	var raw map[string]json.RawMessage
	json.Unmarshal(rec.Body.Bytes(), &raw)
	if rec.Code != http.StatusOK || string(raw["tasks"]) != "[]" {
		t.Fatalf("new board: status %d, tasks %s", rec.Code, raw["tasks"])
	}
}

func TestCreateProjectAcceptsLimits(t *testing.T) {
	h, _ := newTestServer(t)
	for _, body := range []map[string]any{
		{"name": e(100)},
		{"name": "Long description", "description": e(1000)},
	} {
		if rec := serve(h, newReq("POST", "/api/projects", "1", jsonBody(body))); rec.Code != http.StatusCreated {
			t.Fatalf("status %d (body %s)", rec.Code, rec.Body)
		}
	}
}

func TestCreateProjectRefusalsSaveNothing(t *testing.T) {
	h, s := newTestServer(t)
	decodeInto[Project](t, serve(h, newReq("POST", "/api/projects", "1", `{"name":"Website"}`)), http.StatusCreated)
	refused := []struct {
		name, body     string
		field, message string
	}{
		{"same name other case", `{"name":"website"}`, "name", "name is already used by another project"},
		{"seeded name other case", `{"name":"WEBSITE REFRESH"}`, "name", "name is already used by another project"},
		{"name over limit", jsonBody(map[string]any{"name": e(101)}), "name", "name must be 1 to 100 characters"},
		{"name line break", jsonBody(map[string]any{"name": "a\nb"}), "name", "name must be a single line"},
		{"description over limit", jsonBody(map[string]any{"name": "x", "description": e(1001)}), "description", "description must be at most 1000 characters"},
		{"system field", `{"id":9,"name":"x"}`, "", "request body has a field that is not allowed or has the wrong type"},
	}
	for _, tc := range refused {
		t.Run(tc.name, func(t *testing.T) {
			wantError(t, serve(h, newReq("POST", "/api/projects", "1", tc.body)), http.StatusBadRequest, tc.field, tc.message)
			if n := len(snapshot(s).Projects); n != 4 {
				t.Fatalf("project count = %d after a refused create, want 4", n)
			}
		})
	}
}
