package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"
)

var seedTime = time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC)

func snapshot(s *store) data {
	var d data
	s.view(func(x *data) {
		d = data{
			NextID:   x.NextID,
			Projects: slices.Clone(x.Projects),
			Tasks:    slices.Clone(x.Tasks),
			Comments: slices.Clone(x.Comments),
		}
	})
	return d
}

func TestOpenStoreSeedsOnFirstStart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "taskify.json")
	s, err := openStore(path, seedTime)
	if err != nil {
		t.Fatal(err)
	}
	d := snapshot(s)

	var names []string
	for _, p := range d.Projects {
		names = append(names, p.Name)
		if p.CreatedBy != 1 {
			t.Errorf("project %q createdBy = %d, want 1", p.Name, p.CreatedBy)
		}
		if p.CreatedAt != "2026-10-08T09:00:00Z" {
			t.Errorf("project %q createdAt = %q, want 2026-10-08T09:00:00Z", p.Name, p.CreatedAt)
		}
	}
	if want := []string{"Website Refresh", "Mobile App Beta", "Internal Tools"}; !slices.Equal(names, want) {
		t.Fatalf("projects = %q, want %q", names, want)
	}

	assigned := map[int]bool{}
	for _, p := range d.Projects {
		count, statuses, unassigned := 0, map[string]bool{}, false
		for _, task := range d.Tasks {
			if task.ProjectID != p.ID {
				continue
			}
			count++
			statuses[task.Status] = true
			unassigned = unassigned || task.AssigneeID == 0
			assigned[task.AssigneeID] = true
		}
		if count < 4 || count > 6 {
			t.Errorf("%q has %d tasks, want 4–6", p.Name, count)
		}
		for _, st := range []string{"todo", "in_progress", "in_review", "done"} {
			if !statuses[st] {
				t.Errorf("%q has no %s task", p.Name, st)
			}
		}
		if !unassigned {
			t.Errorf("%q has no unassigned task", p.Name)
		}
	}
	for id := 1; id <= 5; id++ {
		if !assigned[id] {
			t.Errorf("user %d is not assigned any sample task", id)
		}
	}
	if len(d.Comments) != 0 {
		t.Errorf("seed has %d comments, want 0", len(d.Comments))
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("file mode = %v, want 0600", fi.Mode().Perm())
	}
}

func TestReopenKeepsDataAndDoesNotSeedAgain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "taskify.json")
	s, err := openStore(path, seedTime)
	if err != nil {
		t.Fatal(err)
	}
	err = s.update(func(d *data) error {
		d.Projects = append(d.Projects, Project{ID: d.newID(), Name: "Extra", CreatedBy: 2, CreatedAt: "2026-10-08T10:00:00Z"})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	again, err := openStore(path, seedTime.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	got := snapshot(again)
	if len(got.Projects) != 4 {
		t.Fatalf("after reopen: %d projects, want 4", len(got.Projects))
	}
	if !reflect.DeepEqual(got, snapshot(s)) {
		t.Fatal("reopened data differs from saved data")
	}
}

func TestCorruptFileIsRefusedAndLeftAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "taskify.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := openStore(path, seedTime); err == nil {
		t.Fatal("want error for corrupt file, got nil")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "{not json" {
		t.Fatalf("corrupt file was changed to %q", b)
	}
}

func TestFailedUpdateLeavesMemoryAndFileUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "taskify.json")
	s, err := openStore(path, seedTime)
	if err != nil {
		t.Fatal(err)
	}
	before := snapshot(s)
	fileBefore, _ := os.ReadFile(path)

	err = s.update(func(d *data) error {
		d.Tasks[0].Title = "changed"
		return errors.New("refused")
	})
	if err == nil {
		t.Fatal("want the fn error back, got nil")
	}
	if !reflect.DeepEqual(snapshot(s), before) {
		t.Fatal("memory changed after a refused update")
	}
	if fileAfter, _ := os.ReadFile(path); string(fileAfter) != string(fileBefore) {
		t.Fatal("file changed after a refused update")
	}
}

func TestFailedSaveRestoresMemory(t *testing.T) {
	dir := t.TempDir()
	s, err := openStore(filepath.Join(dir, "taskify.json"), seedTime)
	if err != nil {
		t.Fatal(err)
	}
	before := snapshot(s)
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}

	err = s.update(func(d *data) error {
		d.Tasks[0].Title = "changed"
		return nil
	})
	if err == nil {
		t.Fatal("want a save error, got nil")
	}
	if !reflect.DeepEqual(snapshot(s), before) {
		t.Fatal("memory kept a change that was never saved")
	}
}

func TestNewIDIsNeverReused(t *testing.T) {
	s, err := openStore(filepath.Join(t.TempDir(), "taskify.json"), seedTime)
	if err != nil {
		t.Fatal(err)
	}
	var first, second int
	addComment := func(id *int) {
		err := s.update(func(d *data) error {
			*id = d.newID()
			d.Comments = append(d.Comments, Comment{ID: *id, TaskID: d.Tasks[0].ID, AuthorID: 1, Text: "x"})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	addComment(&first)
	if err := s.update(func(d *data) error { d.Comments = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	addComment(&second)

	maxSeedID := 0
	for _, task := range snapshot(s).Tasks {
		maxSeedID = max(maxSeedID, task.ID)
	}
	if first <= maxSeedID {
		t.Errorf("first new ID %d is not above the sample IDs (max %d)", first, maxSeedID)
	}
	if second <= first {
		t.Errorf("ID after deleting %d was %d; IDs must only go up", first, second)
	}
}
