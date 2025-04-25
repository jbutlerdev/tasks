package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jbutlerdev/tasks/internal/models"
	"github.com/jbutlerdev/tasks/internal/storage"
)

// API Handlers for Task Lists

// HandleGetAllLists returns all task lists
func HandleGetAllLists(store *storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lists, err := store.GetAllLists()
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "Failed to retrieve lists")
			return
		}

		writeJSON(w, http.StatusOK, lists)
	}
}

// HandleCreateList creates a new task list
func HandleCreateList(store *storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var list models.TaskList

		err := decodeBody(r, &list)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "Invalid list data")
			return
		}

		// Validate list data
		if list.Name == "" {
			writeErrorJSON(w, http.StatusBadRequest, "List name is required")
			return
		}

		// Generate ID if not provided
		if list.ID == "" {
			list.ID = uuid.New().String()
		}

		// Set timestamps
		now := time.Now()
		list.CreatedAt = now
		list.UpdatedAt = now

		// Save the list
		err = store.CreateList(&list)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "Failed to create list")
			return
		}

		writeJSON(w, http.StatusCreated, list)
	}
}

// HandleGetList returns a specific task list
func HandleGetList(store *storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		listID := chi.URLParam(r, "listID")
		if listID == "" {
			writeErrorJSON(w, http.StatusBadRequest, "Missing list ID")
			return
		}

		list, err := store.GetList(listID)
		if err != nil {
			writeErrorJSON(w, http.StatusNotFound, "List not found")
			return
		}

		writeJSON(w, http.StatusOK, list)
	}
}

// HandleUpdateList updates a task list
func HandleUpdateList(store *storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		listID := chi.URLParam(r, "listID")
		if listID == "" {
			writeErrorJSON(w, http.StatusBadRequest, "Missing list ID")
			return
		}

		var list models.TaskList
		err := decodeBody(r, &list)
		if err != nil {
			writeErrorJSON(w, http.StatusBadRequest, "Invalid list data")
			return
		}

		// Verify IDs match
		if list.ID != listID {
			writeErrorJSON(w, http.StatusBadRequest, "List ID in URL does not match list ID in payload")
			return
		}

		// Update timestamps
		list.UpdatedAt = time.Now()

		err = store.UpdateList(&list)
		if err != nil {
			writeErrorJSON(w, http.StatusNotFound, "List not found")
			return
		}

		writeJSON(w, http.StatusOK, list)
	}
}

// HandleDeleteList deletes a task list
func HandleDeleteList(store *storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		listID := chi.URLParam(r, "listID")
		if listID == "" {
			writeErrorJSON(w, http.StatusBadRequest, "Missing list ID")
			return
		}

		err := store.DeleteList(listID)
		if err != nil {
			writeErrorJSON(w, http.StatusNotFound, "List not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// API Handlers for Tasks

// HandleGetAllTasks returns all tasks across all lists
func HandleGetAllTasks(store *storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tasks, err := store.GetAllTasks()
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "Failed to retrieve tasks")
			return
		}

		writeJSON(w, http.StatusOK, tasks)
	}
}

// HandleGetTasksForList returns all tasks in a list
func HandleGetTasksForList(store *storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		listID := chi.URLParam(r, "listID")
		if listID == "" {
			writeErrorJSON(w, http.StatusBadRequest, "Missing list ID")
			return
		}

		tasks, err := store.GetTasksForList(listID)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "Failed to retrieve tasks")
			return
		}

		writeJSON(w, http.StatusOK, tasks)
	}
}

// HandleCreateTask creates a new task in a list
func HandleCreateTask(store *storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		listID := chi.URLParam(r, "listID")
		if listID == "" {
			writeErrorJSON(w, http.StatusBadRequest, "Missing list ID")
			return
		}

		var task models.Task

		// Check if this is a form submission or JSON
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/x-www-form-urlencoded") {
			// Parse form data
			err := r.ParseForm()
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "Failed to parse form data")
				return
			}

			task.Title = r.FormValue("title")
			task.Description = r.FormValue("description")
			task.State = models.TaskState(r.FormValue("state"))

			// Parse due date if provided
			dueDateStr := r.FormValue("due_date")
			if dueDateStr != "" {
				dueDate, err := time.Parse("2006-01-02", dueDateStr)
				if err == nil {
					task.DueDate = &dueDate
				}
			}
		} else {
			// Decode JSON body
			err := decodeBody(r, &task)
			if err != nil {
				writeErrorJSON(w, http.StatusBadRequest, "Invalid task data")
				return
			}
		}

		// Validate task data
		if task.Title == "" {
			writeErrorJSON(w, http.StatusBadRequest, "Task title is required")
			return
		}

		// Set list ID
		task.ListID = listID

		// Generate ID if not provided
		if task.ID == "" {
			task.ID = uuid.New().String()
		}

		// Set timestamps and state
		now := time.Now()
		task.CreatedAt = now
		task.UpdatedAt = now

		// Set default state if not provided
		if task.State == "" {
			task.State = models.TaskStateTodo
		}
		task.StateTime = now

		// Save the task
		err := store.CreateTask(&task)
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "Failed to create task")
			return
		}

		// Check if this is an HTMX request
		if r.Header.Get("HX-Request") == "true" {
			// For HTMX, return the updated tasks container
			tasks, err := store.GetTasksForList(listID)
			if err != nil {
				http.Error(w, "Failed to retrieve tasks", http.StatusInternalServerError)
				return
			}
			html := renderTasksContainer(tasks)
			writeHTMX(w, http.StatusOK, html)
			return
		}

		writeJSON(w, http.StatusCreated, task)
	}
}

// HandleGetTask returns a specific task
func HandleGetTask(store *storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		listID := chi.URLParam(r, "listID")
		taskID := chi.URLParam(r, "taskID")
		if listID == "" || taskID == "" {
			writeErrorJSON(w, http.StatusBadRequest, "Missing list ID or task ID")
			return
		}

		task, err := store.GetTask(listID, taskID)
		if err != nil {
			writeErrorJSON(w, http.StatusNotFound, "Task not found")
			return
		}

		writeJSON(w, http.StatusOK, task)
	}
}

// HandleUpdateTask updates a task
func HandleUpdateTask(store *storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		listID := chi.URLParam(r, "listID")
		taskID := chi.URLParam(r, "taskID")
		if listID == "" || taskID == "" {
			writeErrorJSON(w, http.StatusBadRequest, "Missing list ID or task ID")
			return
		}

		// Try to load existing task, but continue even if not found for new tasks
		existingTask, err := store.GetTask(listID, taskID)
		if err == nil {
			// We found the task, update it
			// Parse updates from request body
			updatedTask := *existingTask // Start with existing data
			
			if err = parseTaskFormOrJSON(r, &updatedTask); err != nil {
				writeErrorJSON(w, http.StatusBadRequest, err.Error())
				return
			}
			
			// Update timestamp and handle state changes
			updatedTask.UpdatedAt = time.Now()
			if updatedTask.State != existingTask.State {
				updatedTask.StateTime = time.Now()
			}
			
			// Handle list changes (move task if needed)
			if updatedTask.ListID != listID {
				_, err = store.MoveTask(listID, taskID, updatedTask.ListID)
			} else {
				err = store.UpdateTask(&updatedTask)
			}
			
			if err != nil {
				writeErrorJSON(w, http.StatusInternalServerError, "Failed to update task: "+err.Error())
				return
			}
			
			// Return response based on request type
			handleTaskResponse(w, r, store, &updatedTask)
			
		} else {
			// Task doesn't exist, create new one
			var newTask models.Task
			newTask.ID = taskID
			newTask.ListID = listID
			
			if err = parseTaskFormOrJSON(r, &newTask); err != nil {
				writeErrorJSON(w, http.StatusBadRequest, err.Error())
				return
			}
			
			// Set timestamps for new task
			now := time.Now()
			newTask.CreatedAt = now
			newTask.UpdatedAt = now
			newTask.StateTime = now
			
			// Ensure state is set
			if newTask.State == "" {
				newTask.State = models.TaskStateTodo
			}
			
			// Save the new task
			err = store.CreateTask(&newTask)
			if err != nil {
				writeErrorJSON(w, http.StatusInternalServerError, "Failed to create task: "+err.Error())
				return
			}
			
			// Return response based on request type
			handleTaskResponse(w, r, store, &newTask)
		}
	}
}

// Helper function to parse task data from either form or JSON
func parseTaskFormOrJSON(r *http.Request, task *models.Task) error {
	contentType := r.Header.Get("Content-Type")
	
	if strings.Contains(contentType, "application/x-www-form-urlencoded") {
		err := r.ParseForm()
		if err != nil {
			return fmt.Errorf("failed to parse form data")
		}
		
		// Get form values, preserving existing values if not in form
		if title := r.FormValue("title"); title != "" {
			task.Title = title
		}
		
		if r.Form.Has("description") {
			task.Description = r.FormValue("description")
		}
		
		if state := r.FormValue("state"); state != "" {
			task.State = models.TaskState(state)
		}
		
		if listID := r.FormValue("list_id"); listID != "" {
			task.ListID = listID
		}
		
		// Handle due date
		if r.Form.Has("due_date") {
			dueDateStr := r.FormValue("due_date")
			if dueDateStr == "clear" || dueDateStr == "" {
				task.DueDate = nil
			} else {
				dueDate, err := time.Parse("2006-01-02", dueDateStr)
				if err == nil {
					task.DueDate = &dueDate
				}
			}
		}
		
	} else {
		// For JSON, we completely override with the new data
		err := decodeBody(r, task)
		if err != nil {
			return fmt.Errorf("invalid JSON data: %w", err)
		}
	}
	
	return nil
}

// Helper function to handle task responses
func handleTaskResponse(w http.ResponseWriter, r *http.Request, store *storage.FileStore, task *models.Task) {
	// Handle HTMX requests differently
	if r.Header.Get("HX-Request") == "true" {
		tasks, err := store.GetTasksForList(task.ListID)
		if err != nil {
			http.Error(w, "Failed to retrieve tasks", http.StatusInternalServerError)
			return
		}
		html := renderTasksContainer(tasks)
		writeHTMX(w, http.StatusOK, html)
	} else {
		// Regular JSON response
		writeJSON(w, http.StatusOK, task)
	}
}

// HandleDeleteTask deletes a task
func HandleDeleteTask(store *storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		listID := chi.URLParam(r, "listID")
		taskID := chi.URLParam(r, "taskID")
		if listID == "" || taskID == "" {
			writeErrorJSON(w, http.StatusBadRequest, "Missing list ID or task ID")
			return
		}

		err := store.DeleteTask(listID, taskID)
		if err != nil {
			writeErrorJSON(w, http.StatusNotFound, "Task not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// Export Handler

// HandleExportMarkdown exports all tasks to markdown
func HandleExportMarkdown(store *storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lists, err := store.GetAllLists()
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "Failed to retrieve lists")
			return
		}

		var buf bytes.Buffer
		buf.WriteString("# Task Lists\n\n")

		for _, list := range lists {
			buf.WriteString(fmt.Sprintf("## %s\n\n", list.Name))
			if list.Description != "" {
				buf.WriteString(fmt.Sprintf("%s\n\n", list.Description))
			}

			tasks, err := store.GetTasksForList(list.ID)
			if err != nil {
				continue
			}

			// Group tasks by state
			tasksByState := make(map[models.TaskState][]models.Task)
			for _, task := range tasks {
				tasksByState[task.State] = append(tasksByState[task.State], task)
			}

			// Write tasks by state
			for _, state := range []models.TaskState{models.TaskStateTodo, models.TaskStateInProgress, models.TaskStateBlocked, models.TaskStateDone} {
				stateTasks := tasksByState[state]
				if len(stateTasks) > 0 {
					buf.WriteString(fmt.Sprintf("### %s\n\n", stateToTitle(state)))
					for _, task := range stateTasks {
						buf.WriteString(fmt.Sprintf("- **%s**", task.Title))
						if task.Description != "" {
							buf.WriteString(fmt.Sprintf(": %s", task.Description))
						}
						if task.DueDate != nil {
							buf.WriteString(fmt.Sprintf(" (Due: %s)", task.DueDate.Format("2006-01-02")))
						}
						buf.WriteString("\n")

						// Add notes if any
						if len(task.Notes) > 0 {
							buf.WriteString("  - Notes:\n")
							for _, note := range task.Notes {
								buf.WriteString(fmt.Sprintf("    - %s\n", note.Content))
							}
						}

						// Add subtasks if any
						if len(task.SubTasks) > 0 {
							buf.WriteString("  - Subtasks:\n")
							for _, subtask := range task.SubTasks {
								buf.WriteString(fmt.Sprintf("    - **%s**", subtask.Title))
								if subtask.Description != "" {
									buf.WriteString(fmt.Sprintf(": %s", subtask.Description))
								}
								buf.WriteString("\n")
							}
						}
					}
					buf.WriteString("\n")
				}
			}
		}

		w.Header().Set("Content-Type", "text/markdown")
		w.Header().Set("Content-Disposition", "attachment; filename=tasks.md")
		w.WriteHeader(http.StatusOK)
		w.Write(buf.Bytes())
	}
}

// Helper to convert state to a title
func stateToTitle(state models.TaskState) string {
	switch state {
	case models.TaskStateTodo:
		return "To Do"
	case models.TaskStateInProgress:
		return "In Progress"
	case models.TaskStateBlocked:
		return "Blocked"
	case models.TaskStateDone:
		return "Done"
	default:
		return string(state)
	}
}

// UI handlers

// HandleHomeUI renders the home page with all tasks
func HandleHomeUI(store *storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tasks, err := store.GetAllTasks()
		if err != nil {
			http.Error(w, "Error loading tasks", http.StatusInternalServerError)
			return
		}

		lists, err := store.GetAllLists()
		if err != nil {
			http.Error(w, "Error loading lists", http.StatusInternalServerError)
			return
		}

		// Create list filter HTML with checkboxes
		var listSelectorHTML bytes.Buffer
		listSelectorHTML.WriteString("<div class=\"list-filter\">")
		listSelectorHTML.WriteString("<h3>Filter by List:</h3>")
		listSelectorHTML.WriteString("<form id=\"list-filter-form\">")
		listSelectorHTML.WriteString("<div class=\"checkbox-group\">")
		listSelectorHTML.WriteString("<label class=\"filter-label\"><input type=\"checkbox\" value=\"all\" checked data-filter-all><span>All Lists</span></label>")
		for _, list := range lists {
			listSelectorHTML.WriteString(fmt.Sprintf("<label class=\"filter-label\"><input type=\"checkbox\" name=\"list\" value=\"%s\" data-list-id=\"%s\"><span>%s</span></label>", 
				list.ID, list.ID, list.Name))
		}
		listSelectorHTML.WriteString("</div>")
		listSelectorHTML.WriteString("</form>")
		listSelectorHTML.WriteString("</div>")

		// In a real app, this would use a template engine
		html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
				<head>
					<title>Task Manager</title>
					<meta charset="UTF-8">
					<meta name="viewport" content="width=device-width, initial-scale=1.0">
					<link rel="icon" href="/static/img/favicon.ico" type="image/x-icon">
					<script src="https://unpkg.com/htmx.org@1.9.2"></script>
					<link rel="stylesheet" href="/static/style.css">
					<script src="/static/app.js" defer></script>
				</head>
				<body>
					<header>
						<h1>Task Manager</h1>
						<nav>
							<a href="/">All Tasks</a>
							<a href="/lists">Task Lists</a>
							<a href="/all-kanban">Kanban View</a>
							<a href="/api/openapi" target="_blank">API Docs</a>
						</nav>
					</header>
					<main>
						<h2>All Tasks</h2>
						<div class="filter-container">
							%s
						</div>
						<div class="tasks-container">
							%s
						</div>
					</main>
				</body>
			</html>
		`, listSelectorHTML.String(), renderAllTasksHTML(tasks, lists))

		writeHTMX(w, http.StatusOK, html)
	}
}

// HandleListsUI renders the lists page
func HandleListsUI(store *storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lists, err := store.GetAllLists()
		if err != nil {
			http.Error(w, "Error loading lists", http.StatusInternalServerError)
			return
		}

		// In a real app, this would use a template engine
		html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
				<head>
					<title>Task Lists</title>
					<meta charset="UTF-8">
					<meta name="viewport" content="width=device-width, initial-scale=1.0">
					<link rel="icon" href="/static/img/favicon.ico" type="image/x-icon">
					<script src="https://unpkg.com/htmx.org@1.9.2"></script>
					<link rel="stylesheet" href="/static/style.css">
					<script src="/static/app.js" defer></script>
				</head>
				<body>
					<header>
						<h1>Task Manager</h1>
						<nav>
							<a href="/">All Tasks</a>
							<a href="/lists">Task Lists</a>
							<a href="/all-kanban">Kanban View</a>
							<a href="/api/openapi" target="_blank">API Docs</a>
						</nav>
					</header>
					<main>
						<h2>Task Lists</h2>
						<div class="lists-container">
							%s
						</div>
						<div class="new-list-form">
							<h3>Create New List</h3>
							<form hx-post="/api/lists" hx-target=".lists-container" hx-swap="innerHTML" enctype="application/x-www-form-urlencoded">
								<div>
									<label for="name">Name:</label>
									<input type="text" id="name" name="name" required>
								</div>
								<div>
									<label for="description">Description:</label>
									<textarea id="description" name="description"></textarea>
								</div>
								<button type="submit">Create List</button>
							</form>
						</div>
					</main>
				</body>
			</html>
		`, renderListsHTML(lists))

		writeHTMX(w, http.StatusOK, html)
	}
}

// HandleListUI renders a single list with its tasks
func HandleListUI(store *storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		listID := chi.URLParam(r, "listID")
		if listID == "" {
			http.Error(w, "Missing list ID", http.StatusBadRequest)
			return
		}

		list, err := store.GetList(listID)
		if err != nil {
			http.Error(w, "List not found", http.StatusNotFound)
			return
		}

		tasks, err := store.GetTasksForList(listID)
		if err != nil {
			http.Error(w, "Error loading tasks", http.StatusInternalServerError)
			return
		}

		// In a real app, this would use a template engine
		html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
				<head>
					<title>%s</title>
					<meta charset="UTF-8">
					<meta name="viewport" content="width=device-width, initial-scale=1.0">
					<link rel="icon" href="/static/img/favicon.ico" type="image/x-icon">
					<script src="https://unpkg.com/htmx.org@1.9.2"></script>
					<link rel="stylesheet" href="/static/style.css">
					<script src="/static/app.js" defer></script>
				</head>
				<body>
					<header>
						<h1>Task Manager</h1>
						<nav>
							<a href="/">All Tasks</a>
							<a href="/lists">Task Lists</a>
							<a href="/kanban/%s">Kanban View</a>
							<a href="/api/openapi" target="_blank">API Docs</a>
						</nav>
					</header>
					<main>
						<h2>%s</h2>
						<p>%s</p>
						<div class="tasks-container">
							%s
						</div>
						<div class="new-task-form">
							<h3>Create New Task</h3>
							<form hx-post="/api/lists/%s/tasks" hx-target=".tasks-container" hx-swap="outerHTML" enctype="application/x-www-form-urlencoded">
								<div>
									<label for="title">Title:</label>
									<input type="text" id="title" name="title" required>
								</div>
								<div>
									<label for="description">Description:</label>
									<textarea id="description" name="description"></textarea>
								</div>
								<div>
									<label for="state">State:</label>
									<select id="state" name="state">
										<option value="todo">Todo</option>
										<option value="in_progress">In Progress</option>
										<option value="blocked">Blocked</option>
										<option value="done">Done</option>
									</select>
								</div>
								<div>
									<label for="due_date">Due Date:</label>
									<input type="date" id="due_date" name="due_date">
								</div>
								<button type="submit">Create Task</button>
							</form>
						</div>
					</main>

					<!-- Task edit modal -->
					<div id="task-edit-modal" class="modal">
						<div class="modal-content">
							<span class="close">&times;</span>
							<h2>Edit Task</h2>
							<form id="edit-task-form" enctype="application/x-www-form-urlencoded">
								<input type="hidden" id="edit-task-id" name="id">
								<div>
									<label for="edit-title">Title:</label>
									<input type="text" id="edit-title" name="title" required>
								</div>
								<div>
									<label for="edit-description">Description:</label>
									<textarea id="edit-description" name="description"></textarea>
								</div>
								<div>
									<label for="edit-state">State:</label>
									<select id="edit-state" name="state">
										<option value="todo">Todo</option>
										<option value="in_progress">In Progress</option>
										<option value="blocked">Blocked</option>
										<option value="done">Done</option>
									</select>
								</div>
								<div>
									<label for="edit-due-date">Due Date:</label>
									<input type="date" id="edit-due-date" name="due_date">
									<button type="button" id="clear-due-date">Clear</button>
								</div>
								<div>
									<label for="edit-list-id">List:</label>
									<select id="edit-list-id" name="list_id">
										<!-- Will be populated by JavaScript -->
									</select>
								</div>
								<button type="submit">Update Task</button>
							</form>
						</div>
					</div>
				</body>
			</html>
		`, list.Name, listID, list.Name, list.Description, renderTasksHTML(tasks), listID)

		writeHTMX(w, http.StatusOK, html)
	}
}

// HandleKanbanUI renders a kanban view of a list's tasks
func HandleKanbanUI(store *storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		listID := chi.URLParam(r, "listID")
		if listID == "" {
			http.Error(w, "Missing list ID", http.StatusBadRequest)
			return
		}

		list, err := store.GetList(listID)
		if err != nil {
			http.Error(w, "List not found", http.StatusNotFound)
			return
		}

		tasks, err := store.GetTasksForList(listID)
		if err != nil {
			http.Error(w, "Error loading tasks", http.StatusInternalServerError)
			return
		}

		// Group tasks by state
		tasksByState := make(map[models.TaskState][]models.Task)
		for _, task := range tasks {
			tasksByState[task.State] = append(tasksByState[task.State], task)
		}

		// In a real app, this would use a template engine
		html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
				<head>
					<title>Kanban - %s</title>
					<meta charset="UTF-8">
					<meta name="viewport" content="width=device-width, initial-scale=1.0">
					<link rel="icon" href="/static/img/favicon.ico" type="image/x-icon">
					<script src="https://unpkg.com/htmx.org@1.9.2"></script>
					<link rel="stylesheet" href="/static/style.css">
					<script src="/static/app.js" defer></script>
				</head>
				<body>
					<header>
						<h1>Task Manager</h1>
						<nav>
							<a href="/">All Tasks</a>
							<a href="/lists">Task Lists</a>
							<a href="/all-kanban">All Kanban</a>
							<a href="/lists/%s">List View</a>
							<a href="/api/openapi" target="_blank">API Docs</a>
						</nav>
					</header>
					<main>
						<h2>Kanban Board - %s</h2>
						<div class="kanban-board">
							<div class="kanban-column">
								<h3>Todo</h3>
								<div class="kanban-tasks">
									%s
								</div>
							</div>
							<div class="kanban-column">
								<h3>In Progress</h3>
								<div class="kanban-tasks">
									%s
								</div>
							</div>
							<div class="kanban-column">
								<h3>Blocked</h3>
								<div class="kanban-tasks">
									%s
								</div>
							</div>
							<div class="kanban-column">
								<h3>Done</h3>
								<div class="kanban-tasks">
									%s
								</div>
							</div>
						</div>
					</main>

					<!-- Task edit modal -->
					<div id="task-edit-modal" class="modal">
						<div class="modal-content">
							<span class="close">&times;</span>
							<h2>Edit Task</h2>
							<form id="edit-task-form" enctype="application/x-www-form-urlencoded">
								<input type="hidden" id="edit-task-id" name="id">
								<div>
									<label for="edit-title">Title:</label>
									<input type="text" id="edit-title" name="title" required>
								</div>
								<div>
									<label for="edit-description">Description:</label>
									<textarea id="edit-description" name="description"></textarea>
								</div>
								<div>
									<label for="edit-state">State:</label>
									<select id="edit-state" name="state">
										<option value="todo">Todo</option>
										<option value="in_progress">In Progress</option>
										<option value="blocked">Blocked</option>
										<option value="done">Done</option>
									</select>
								</div>
								<div>
									<label for="edit-due-date">Due Date:</label>
									<input type="date" id="edit-due-date" name="due_date">
									<button type="button" id="clear-due-date">Clear</button>
								</div>
								<div>
									<label for="edit-list-id">List:</label>
									<select id="edit-list-id" name="list_id">
										<!-- Will be populated by JavaScript -->
									</select>
								</div>
								<button type="submit">Update Task</button>
							</form>
						</div>
					</div>
				</body>
			</html>
		`, list.Name, listID, list.Name, 
			renderKanbanTasksHTML(tasksByState[models.TaskStateTodo]), 
			renderKanbanTasksHTML(tasksByState[models.TaskStateInProgress]),
			renderKanbanTasksHTML(tasksByState[models.TaskStateBlocked]),
			renderKanbanTasksHTML(tasksByState[models.TaskStateDone]))

		writeHTMX(w, http.StatusOK, html)
	}
}

// HTML rendering helpers

// renderAllTasksHTML renders all tasks with list names
func renderAllTasksHTML(tasks []models.Task, lists []models.TaskList) string {
	if len(tasks) == 0 {
		return "<p>No tasks found</p>"
	}

	// Create a map of list IDs to names
	listNames := make(map[string]string)
	for _, list := range lists {
		listNames[list.ID] = list.Name
	}

	var buf bytes.Buffer
	buf.WriteString("<div class=\"tasks\">")
	for _, task := range tasks {
		listName := listNames[task.ListID]
		buf.WriteString(fmt.Sprintf(`
			<div class="task task-state-%s" data-task-id="%s" data-list-id="%s">
				<div class="task-header">
					<h3>%s</h3>
					<span class="task-list">%s</span>
				</div>
				<div class="task-body">
					<p>%s</p>
					<div class="task-meta">
						<span class="task-state">%s</span>
						%s
					</div>
				</div>
			</div>
		`, task.State, task.ID, task.ListID, task.Title, listName, task.Description, stateToTitle(task.State), renderDueDate(task.DueDate)))
	}
	buf.WriteString("</div>")
	return buf.String()
}

// renderTasksHTML renders tasks for a specific list
func renderTasksHTML(tasks []models.Task) string {
	if len(tasks) == 0 {
		return "<p>No tasks found</p>"
	}

	var buf bytes.Buffer
	buf.WriteString("<div class=\"tasks\">")
	for _, task := range tasks {
		buf.WriteString(fmt.Sprintf(`
			<div class="task task-state-%s" data-task-id="%s" data-list-id="%s">
				<div class="task-header">
					<h3>%s</h3>
				</div>
				<div class="task-body">
					<p>%s</p>
					<div class="task-meta">
						<span class="task-state">%s</span>
						%s
					</div>
				</div>
			</div>
		`, task.State, task.ID, task.ListID, task.Title, task.Description, stateToTitle(task.State), renderDueDate(task.DueDate)))
	}
	buf.WriteString("</div>")
	return buf.String()
}

// renderTasksContainer wraps the tasks HTML in a container
func renderTasksContainer(tasks []models.Task) string {
	return fmt.Sprintf(`
		<div class="tasks-container">
			%s
		</div>
	`, renderTasksHTML(tasks))
}

// renderKanbanTasksHTML renders tasks for a kanban column
func renderKanbanTasksHTML(tasks []models.Task) string {
	if len(tasks) == 0 {
		return "<p class=\"empty-column\">No tasks</p>"
	}

	var buf bytes.Buffer
	for _, task := range tasks {
		buf.WriteString(fmt.Sprintf(`
			<div class="kanban-task" data-task-id="%s" data-list-id="%s">
				<h4>%s</h4>
				<p>%s</p>
				<div class="task-meta">
					%s
				</div>
			</div>
		`, task.ID, task.ListID, task.Title, task.Description, renderDueDate(task.DueDate)))
	}
	return buf.String()
}

// renderListsHTML renders all lists
func renderListsHTML(lists []models.TaskList) string {
	if len(lists) == 0 {
		return "<p>No lists found</p>"
	}

	var buf bytes.Buffer
	buf.WriteString("<div class=\"lists\">")
	for _, list := range lists {
		buf.WriteString(fmt.Sprintf(`
			<div class="list">
				<div class="list-header">
					<h3><a href="/lists/%s">%s</a></h3>
				</div>
				<div class="list-body">
					<p>%s</p>
					<div class="list-actions">
						<a href="/lists/%s" class="button">View Tasks</a>
						<a href="/kanban/%s" class="button">Kanban View</a>
					</div>
				</div>
			</div>
		`, list.ID, list.Name, list.Description, list.ID, list.ID))
	}
	buf.WriteString("</div>")
	return buf.String()
}

// renderDueDate formats a due date or returns empty string
func renderDueDate(dueDate *time.Time) string {
	if dueDate == nil {
		return ""
	}
	return fmt.Sprintf("<span class=\"task-due-date\">Due: %s</span>", dueDate.Format("2006-01-02"))
}

// HandleAllKanbanUI renders a kanban view of all tasks across all lists
func HandleAllKanbanUI(store *storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get all lists for the header links
		lists, err := store.GetAllLists()
		if err != nil {
			http.Error(w, "Error loading lists", http.StatusInternalServerError)
			return
		}

		// Get all tasks
		tasks, err := store.GetAllTasks()
		if err != nil {
			http.Error(w, "Error loading tasks", http.StatusInternalServerError)
			return
		}

		// Get list names for task display
		listNames := make(map[string]string)
		for _, list := range lists {
			listNames[list.ID] = list.Name
		}
		
		// Group tasks by state
		tasksByState := make(map[models.TaskState][]models.Task)
		for _, task := range tasks {
			tasksByState[task.State] = append(tasksByState[task.State], task)
		}

		// Create list filter HTML with checkboxes
		var listSelectorHTML bytes.Buffer
		listSelectorHTML.WriteString("<div class=\"list-filter\">")
		listSelectorHTML.WriteString("<h3>Filter by List:</h3>")
		listSelectorHTML.WriteString("<form id=\"list-filter-form\">")
		listSelectorHTML.WriteString("<div class=\"checkbox-group\">")
		listSelectorHTML.WriteString("<label class=\"filter-label\"><input type=\"checkbox\" value=\"all\" checked data-filter-all><span>All Lists</span></label>")
		for _, list := range lists {
			listSelectorHTML.WriteString(fmt.Sprintf("<label class=\"filter-label\"><input type=\"checkbox\" name=\"list\" value=\"%s\" data-list-id=\"%s\"><span>%s</span></label>", 
				list.ID, list.ID, list.Name))
		}
		listSelectorHTML.WriteString("</div>")
		listSelectorHTML.WriteString("</form>")
		listSelectorHTML.WriteString("</div>")

		// In a real app, this would use a template engine
		html := fmt.Sprintf(`
			<!DOCTYPE html>
			<html>
				<head>
					<title>Kanban - All Tasks</title>
					<meta charset="UTF-8">
					<meta name="viewport" content="width=device-width, initial-scale=1.0">
					<link rel="icon" href="/static/img/favicon.ico" type="image/x-icon">
					<script src="https://unpkg.com/htmx.org@1.9.2"></script>
					<link rel="stylesheet" href="/static/style.css">
					<script src="/static/app.js" defer></script>
				</head>
				<body>
					<header>
						<h1>Task Manager</h1>
						<nav>
							<a href="/">All Tasks</a>
							<a href="/lists">Task Lists</a>
							<a href="/all-kanban">Kanban View</a>
							<a href="/api/openapi" target="_blank">API Docs</a>
						</nav>
					</header>
					<main>
						<h2>Kanban Board - All Tasks</h2>
						%s
						<div class="kanban-board">
							<div class="kanban-column">
								<h3>Todo</h3>
								<div class="kanban-tasks">
									%s
								</div>
							</div>
							<div class="kanban-column">
								<h3>In Progress</h3>
								<div class="kanban-tasks">
									%s
								</div>
							</div>
							<div class="kanban-column">
								<h3>Blocked</h3>
								<div class="kanban-tasks">
									%s
								</div>
							</div>
							<div class="kanban-column">
								<h3>Done</h3>
								<div class="kanban-tasks">
									%s
								</div>
							</div>
						</div>
					</main>

					<!-- Task edit modal -->
					<div id="task-edit-modal" class="modal">
						<div class="modal-content">
							<span class="close">&times;</span>
							<h2>Edit Task</h2>
							<form id="edit-task-form" enctype="application/x-www-form-urlencoded">
								<input type="hidden" id="edit-task-id" name="id">
								<div>
									<label for="edit-title">Title:</label>
									<input type="text" id="edit-title" name="title" required>
								</div>
								<div>
									<label for="edit-description">Description:</label>
									<textarea id="edit-description" name="description"></textarea>
								</div>
								<div>
									<label for="edit-state">State:</label>
									<select id="edit-state" name="state">
										<option value="todo">Todo</option>
										<option value="in_progress">In Progress</option>
										<option value="blocked">Blocked</option>
										<option value="done">Done</option>
									</select>
								</div>
								<div>
									<label for="edit-due-date">Due Date:</label>
									<input type="date" id="edit-due-date" name="due_date">
									<button type="button" id="clear-due-date">Clear</button>
								</div>
								<div>
									<label for="edit-list-id">List:</label>
									<select id="edit-list-id" name="list_id">
										<!-- Will be populated by JavaScript -->
									</select>
								</div>
								<button type="submit">Update Task</button>
							</form>
						</div>
					</div>
				</body>
			</html>
		`, listSelectorHTML.String(),
			renderKanbanTasksHTML(tasksByState[models.TaskStateTodo]), 
			renderKanbanTasksHTML(tasksByState[models.TaskStateInProgress]),
			renderKanbanTasksHTML(tasksByState[models.TaskStateBlocked]),
			renderKanbanTasksHTML(tasksByState[models.TaskStateDone]))

		writeHTMX(w, http.StatusOK, html)
	}
}

// Utility functions

// decodeBody decodes a request body into a struct
func decodeBody(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeErrorJSON writes a JSON error response
func writeErrorJSON(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// writeHTMX writes an HTMX response
func writeHTMX(w http.ResponseWriter, status int, content string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprint(w, content)
}

// HandleOpenAPISpec generates and returns the OpenAPI specification for the API
func HandleOpenAPISpec(store *storage.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Generate an API spec manually with just JSON marshaling
		spec := map[string]interface{}{
			"openapi": "3.0.0",
			"info": map[string]interface{}{
				"title":       "Tasks API",
				"description": "API for task management system",
				"version":     "1.0.0",
				"contact": map[string]string{
					"name": "Developer",
				},
			},
			"servers": []map[string]string{
				{"url": "/", "description": "Current server"},
			},
			"paths": map[string]interface{}{
				"/api/lists": map[string]interface{}{
					"get": map[string]interface{}{
						"summary":     "Get all lists",
						"description": "Returns all task lists",
						"operationId": "getAllLists",
						"responses": map[string]interface{}{
							"200": map[string]interface{}{
								"description": "Successful operation",
								"content": map[string]interface{}{
									"application/json": map[string]interface{}{
										"schema": map[string]interface{}{
											"type":  "array",
											"items": map[string]string{"$ref": "#/components/schemas/TaskList"},
										},
									},
								},
							},
						},
					},
					"post": map[string]interface{}{
						"summary":     "Create a new list",
						"description": "Creates a new task list",
						"operationId": "createList",
						"requestBody": map[string]interface{}{
							"required": true,
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{"$ref": "#/components/schemas/TaskList"},
								},
							},
						},
						"responses": map[string]interface{}{
							"201": map[string]interface{}{
								"description": "List created",
								"content": map[string]interface{}{
									"application/json": map[string]interface{}{
										"schema": map[string]string{"$ref": "#/components/schemas/TaskList"},
									},
								},
							},
						},
					},
				},
				"/api/lists/{listID}": map[string]interface{}{
					"parameters": []map[string]interface{}{
						{
							"name":        "listID",
							"in":          "path",
							"required":    true,
							"description": "ID of the task list",
							"schema":      map[string]string{"type": "string"},
						},
					},
					"get": map[string]interface{}{
						"summary":     "Get a task list",
						"description": "Returns a task list by ID",
						"operationId": "getList",
						"responses": map[string]interface{}{
							"200": map[string]interface{}{
								"description": "Successful operation",
								"content": map[string]interface{}{
									"application/json": map[string]interface{}{
										"schema": map[string]string{"$ref": "#/components/schemas/TaskList"},
									},
								},
							},
							"404": map[string]interface{}{
								"description": "List not found",
							},
						},
					},
					"put": map[string]interface{}{
						"summary":     "Update a task list",
						"description": "Updates a task list by ID",
						"operationId": "updateList",
						"requestBody": map[string]interface{}{
							"required": true,
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{"$ref": "#/components/schemas/TaskList"},
								},
							},
						},
						"responses": map[string]interface{}{
							"200": map[string]interface{}{
								"description": "List updated",
							},
							"404": map[string]interface{}{
								"description": "List not found",
							},
						},
					},
					"delete": map[string]interface{}{
						"summary":     "Delete a task list",
						"description": "Deletes a task list by ID",
						"operationId": "deleteList",
						"responses": map[string]interface{}{
							"204": map[string]interface{}{
								"description": "List deleted",
							},
							"404": map[string]interface{}{
								"description": "List not found",
							},
						},
					},
				},
				"/api/lists/{listID}/tasks": map[string]interface{}{
					"parameters": []map[string]interface{}{
						{
							"name":        "listID",
							"in":          "path",
							"required":    true,
							"description": "ID of the task list",
							"schema":      map[string]string{"type": "string"},
						},
					},
					"get": map[string]interface{}{
						"summary":     "Get tasks for a list",
						"description": "Returns all tasks in a specific list",
						"operationId": "getTasksForList",
						"responses": map[string]interface{}{
							"200": map[string]interface{}{
								"description": "Successful operation",
								"content": map[string]interface{}{
									"application/json": map[string]interface{}{
										"schema": map[string]interface{}{
											"type":  "array",
											"items": map[string]string{"$ref": "#/components/schemas/Task"},
										},
									},
								},
							},
						},
					},
					"post": map[string]interface{}{
						"summary":     "Create a task in a list",
						"description": "Creates a new task in the specified list",
						"operationId": "createTask",
						"requestBody": map[string]interface{}{
							"required": true,
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{"$ref": "#/components/schemas/Task"},
								},
							},
						},
						"responses": map[string]interface{}{
							"201": map[string]interface{}{
								"description": "Task created",
								"content": map[string]interface{}{
									"application/json": map[string]interface{}{
										"schema": map[string]string{"$ref": "#/components/schemas/Task"},
									},
								},
							},
						},
					},
				},
				"/api/tasks": map[string]interface{}{
					"get": map[string]interface{}{
						"summary":     "Get all tasks",
						"description": "Returns all tasks across all lists",
						"operationId": "getAllTasks",
						"responses": map[string]interface{}{
							"200": map[string]interface{}{
								"description": "Successful operation",
								"content": map[string]interface{}{
									"application/json": map[string]interface{}{
										"schema": map[string]interface{}{
											"type":  "array",
											"items": map[string]string{"$ref": "#/components/schemas/Task"},
										},
									},
								},
							},
						},
					},
				},
				"/api/tasks/{listID}/{taskID}": map[string]interface{}{
					"parameters": []map[string]interface{}{
						{
							"name":        "listID",
							"in":          "path",
							"required":    true,
							"description": "ID of the task list",
							"schema":      map[string]string{"type": "string"},
						},
						{
							"name":        "taskID",
							"in":          "path",
							"required":    true,
							"description": "ID of the task",
							"schema":      map[string]string{"type": "string"},
						},
					},
					"get": map[string]interface{}{
						"summary":     "Get a task",
						"description": "Returns a task by ID",
						"operationId": "getTask",
						"responses": map[string]interface{}{
							"200": map[string]interface{}{
								"description": "Successful operation",
								"content": map[string]interface{}{
									"application/json": map[string]interface{}{
										"schema": map[string]string{"$ref": "#/components/schemas/Task"},
									},
								},
							},
							"404": map[string]interface{}{
								"description": "Task not found",
							},
						},
					},
					"put": map[string]interface{}{
						"summary":     "Update a task",
						"description": "Updates a task by ID",
						"operationId": "updateTask",
						"requestBody": map[string]interface{}{
							"required": true,
							"content": map[string]interface{}{
								"application/json": map[string]interface{}{
									"schema": map[string]string{"$ref": "#/components/schemas/Task"},
								},
							},
						},
						"responses": map[string]interface{}{
							"200": map[string]interface{}{
								"description": "Task updated",
								"content": map[string]interface{}{
									"application/json": map[string]interface{}{
										"schema": map[string]string{"$ref": "#/components/schemas/Task"},
									},
								},
							},
							"404": map[string]interface{}{
								"description": "Task not found",
							},
						},
					},
					"delete": map[string]interface{}{
						"summary":     "Delete a task",
						"description": "Deletes a task by ID",
						"operationId": "deleteTask",
						"responses": map[string]interface{}{
							"204": map[string]interface{}{
								"description": "Task deleted",
							},
							"404": map[string]interface{}{
								"description": "Task not found",
							},
						},
					},
				},
				"/api/export": map[string]interface{}{
					"get": map[string]interface{}{
						"summary":     "Export to markdown",
						"description": "Exports all tasks to markdown format",
						"operationId": "exportMarkdown",
						"responses": map[string]interface{}{
							"200": map[string]interface{}{
								"description": "Successful operation",
								"content": map[string]interface{}{
									"text/markdown": map[string]interface{}{
										"schema": map[string]string{"type": "string"},
									},
								},
							},
						},
					},
				},
				"/api/openapi": map[string]interface{}{
					"get": map[string]interface{}{
						"summary":     "Get OpenAPI specification",
						"description": "Returns the OpenAPI specification for this API",
						"operationId": "getOpenAPISpec",
						"responses": map[string]interface{}{
							"200": map[string]interface{}{
								"description": "Successful operation",
								"content": map[string]interface{}{
									"application/json": map[string]interface{}{
										"schema": map[string]string{"type": "object"},
									},
								},
							},
						},
					},
				},
			},
			"components": map[string]interface{}{
				"schemas": map[string]interface{}{
					"Task": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"id": map[string]string{
								"type":        "string",
								"description": "Unique task identifier",
							},
							"title": map[string]string{
								"type":        "string",
								"description": "Task title",
							},
							"description": map[string]string{
								"type":        "string",
								"description": "Task description",
							},
							"list_id": map[string]string{
								"type":        "string",
								"description": "ID of the list the task belongs to",
							},
							"state": map[string]interface{}{
								"type":        "string",
								"description": "Task state",
								"enum":        []string{"todo", "in_progress", "blocked", "done"},
							},
							"state_time": map[string]string{
								"type":        "string",
								"format":      "date-time",
								"description": "Time when the current state was set",
							},
							"due_date": map[string]interface{}{
								"type":        "string",
								"format":      "date-time",
								"description": "Task due date",
								"nullable":    true,
							},
							"created_at": map[string]string{
								"type":        "string",
								"format":      "date-time",
								"description": "Creation time",
							},
							"updated_at": map[string]string{
								"type":        "string",
								"format":      "date-time",
								"description": "Last update time",
							},
							"notes": map[string]interface{}{
								"type":        "array",
								"description": "Task notes",
								"items":       map[string]string{"$ref": "#/components/schemas/Note"},
							},
							"sub_tasks": map[string]interface{}{
								"type":        "array",
								"description": "Sub-tasks",
								"items":       map[string]string{"$ref": "#/components/schemas/Task"},
							},
						},
						"required": []string{"id", "title", "list_id", "state", "state_time", "created_at", "updated_at"},
					},
					"Note": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"id": map[string]string{
								"type":        "string",
								"description": "Note identifier",
							},
							"content": map[string]string{
								"type":        "string",
								"description": "Note content",
							},
							"created_at": map[string]string{
								"type":        "string",
								"format":      "date-time",
								"description": "Creation time",
							},
							"updated_at": map[string]string{
								"type":        "string",
								"format":      "date-time",
								"description": "Last update time",
							},
						},
						"required": []string{"id", "content", "created_at", "updated_at"},
					},
					"TaskList": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"id": map[string]string{
								"type":        "string",
								"description": "Task list identifier",
							},
							"name": map[string]string{
								"type":        "string",
								"description": "Task list name",
							},
							"description": map[string]string{
								"type":        "string",
								"description": "Task list description",
							},
							"created_at": map[string]string{
								"type":        "string",
								"format":      "date-time",
								"description": "Creation time",
							},
							"updated_at": map[string]string{
								"type":        "string",
								"format":      "date-time",
								"description": "Last update time",
							},
						},
						"required": []string{"id", "name", "created_at", "updated_at"},
					},
				},
			},
		}

		// Convert to JSON
		jsonData, err := json.MarshalIndent(spec, "", "  ")
		if err != nil {
			writeErrorJSON(w, http.StatusInternalServerError, "Failed to generate OpenAPI spec")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonData)
	}
}