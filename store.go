package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type Project struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedBy   int    `json:"createdBy"`
	CreatedAt   string `json:"createdAt"`
}

type Task struct {
	ID          int    `json:"id"`
	ProjectID   int    `json:"projectId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	AssigneeID  int    `json:"assigneeId"`
	CreatedBy   int    `json:"createdBy"`
	CreatedAt   string `json:"createdAt"`
}

type Comment struct {
	ID        int    `json:"id"`
	TaskID    int    `json:"taskId"`
	AuthorID  int    `json:"authorId"`
	Text      string `json:"text"`
	CreatedAt string `json:"createdAt"`
}

type data struct {
	NextID   int       `json:"nextId"`
	Projects []Project `json:"projects"`
	Tasks    []Task    `json:"tasks"`
	Comments []Comment `json:"comments"`
}

// ponytail: one mutex and a full-file rewrite per change; fine to a few thousand tasks (research R3).
type store struct {
	mu    sync.Mutex
	path  string
	data  data
	saved []byte
}

func openStore(path string, now time.Time) (*store, error) {
	s := &store{path: path}
	b, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		s.data = seedData(now)
		if err := s.save(); err != nil {
			return nil, err
		}
		return s, nil
	case err != nil:
		return nil, err
	}
	if err := json.Unmarshal(b, &s.data); err != nil {
		return nil, fmt.Errorf("%s is not a valid Taskify data file: %w", path, err)
	}
	s.saved = b
	return s, nil
}

// save writes a temp file and renames it over the real one; rename is atomic, so a crash
// leaves either the old file or the new one, never half of one.
func (s *store) save() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".taskify-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(f.Name(), s.path)
	}
	if err == nil {
		s.saved = b
	}
	return err
}

func (s *store) view(fn func(d *data)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.data)
}

func (s *store) update(fn func(d *data) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := fn(&s.data)
	if err == nil {
		err = s.save()
	}
	if err != nil {
		// Rebuild from the last saved bytes so memory never holds a change the file lacks.
		// A fresh value is required: unmarshalling into s.data would merge old slices.
		var d data
		if json.Unmarshal(s.saved, &d) == nil {
			s.data = d
		}
	}
	return err
}

func (d *data) newID() int {
	id := d.NextID
	d.NextID++
	return id
}

var errNotFound = errors.New("not found")

func (d *data) project(id int) *Project {
	for i := range d.Projects {
		if d.Projects[i].ID == id {
			return &d.Projects[i]
		}
	}
	return nil
}

// task only matches inside project pid, so a task ID under the wrong project is a 404.
func (d *data) task(pid, tid int) *Task {
	for i := range d.Tasks {
		if d.Tasks[i].ID == tid && d.Tasks[i].ProjectID == pid {
			return &d.Tasks[i]
		}
	}
	return nil
}
