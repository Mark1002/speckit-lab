package main

import "time"

var users = []User{
	{1, "Avery Chen", "product_manager"},
	{2, "Blake Rivera", "engineer"},
	{3, "Casey Patel", "engineer"},
	{4, "Devon Kim", "engineer"},
	{5, "Emery Novak", "engineer"},
}

func seedData(now time.Time) data {
	at := now.UTC().Format(time.RFC3339)
	d := data{NextID: 1}
	for _, name := range []string{"Website Refresh", "Mobile App Beta", "Internal Tools"} {
		d.Projects = append(d.Projects, Project{ID: d.newID(), Name: name, CreatedBy: 1, CreatedAt: at})
	}
	for _, t := range []struct {
		project       int
		title, status string
		assignee      int
	}{
		{1, "Draft new homepage copy", "todo", 1},
		{1, "Pick a color palette", "todo", 0},
		{1, "Build responsive header", "in_progress", 2},
		{1, "Review navigation changes", "in_review", 3},
		{1, "Set up analytics", "done", 2},
		{2, "Write beta test plan", "todo", 1},
		{2, "Collect tester feedback", "todo", 0},
		{2, "Fix login screen layout", "in_progress", 4},
		{2, "Add crash reporting", "in_review", 5},
		{2, "Ship first beta build", "done", 4},
		{3, "Automate weekly report", "todo", 5},
		{3, "Clean up old scripts", "in_progress", 0},
		{3, "Document deploy steps", "in_review", 3},
		{3, "Upgrade build server", "done", 2},
	} {
		d.Tasks = append(d.Tasks, Task{
			ID: d.newID(), ProjectID: t.project, Title: t.title, Status: t.status,
			AssigneeID: t.assignee, CreatedBy: 1, CreatedAt: at,
		})
	}
	return d
}
