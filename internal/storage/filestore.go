package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jbutlerdev/tasks/internal/models"
)

// TaskStore defines the interface for task storage
type TaskStore interface {
	// List operations
	GetAllLists() ([]models.TaskList, error)
	GetList(id string) (*models.TaskList, error)
	CreateList(list *models.TaskList) error
	UpdateList(list *models.TaskList) error
	DeleteList(id string) error
	
	// Task operations
	GetAllTasks() ([]models.Task, error)
	GetTasksByList(listID string) ([]models.Task, error)
	GetTask(listID, taskID string) (*models.Task, error)
	CreateTask(task *models.Task) error
	UpdateTask(task *models.Task) error
	DeleteTask(listID, taskID string) error
}

type FileStore struct {
	baseDir string
	mutex   *sync.RWMutex
}

// NewFileStore creates a new file-based storage system
func NewFileStore(baseDir string) (*FileStore, error) {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Create lists directory if it doesn't exist
	listsDir := filepath.Join(baseDir, "lists")
	if err := os.MkdirAll(listsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create lists directory: %w", err)
	}

	return &FileStore{
		baseDir: baseDir,
		mutex:   &sync.RWMutex{},
	}, nil
}

// Task List Methods

// GetAllLists returns all task lists
func (fs *FileStore) GetAllLists() ([]models.TaskList, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	listsDir := filepath.Join(fs.baseDir, "lists")
	files, err := os.ReadDir(listsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read lists directory: %w", err)
	}

	var lists []models.TaskList
	for _, file := range files {
		if file.IsDir() {
			// Each directory represents a list
			listPath := filepath.Join(listsDir, file.Name(), "list.json")
			
			// Read list file
			data, err := os.ReadFile(listPath)
			if err != nil {
				// Skip if list file cannot be read
				continue
			}

			var list models.TaskList
			if err := json.Unmarshal(data, &list); err != nil {
				// Skip if list file cannot be parsed
				continue
			}

			lists = append(lists, list)
		}
	}

	return lists, nil
}

// GetList returns a single task list by ID
func (fs *FileStore) GetList(id string) (*models.TaskList, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	listPath := filepath.Join(fs.baseDir, "lists", id, "list.json")
	data, err := os.ReadFile(listPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("list not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read list: %w", err)
	}

	var list models.TaskList
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("failed to parse list: %w", err)
	}

	return &list, nil
}

// CreateList creates a new task list
func (fs *FileStore) CreateList(list *models.TaskList) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// Set timestamps
	now := time.Now()
	list.CreatedAt = now
	list.UpdatedAt = now

	// Create list directory
	listDir := filepath.Join(fs.baseDir, "lists", list.ID)
	if err := os.MkdirAll(listDir, 0755); err != nil {
		return fmt.Errorf("failed to create list directory: %w", err)
	}

	// Write list file
	listPath := filepath.Join(listDir, "list.json")
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize list: %w", err)
	}

	if err := os.WriteFile(listPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write list file: %w", err)
	}

	// Create tasks directory
	tasksDir := filepath.Join(listDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return fmt.Errorf("failed to create tasks directory: %w", err)
	}

	return nil
}

// UpdateList updates an existing task list
func (fs *FileStore) UpdateList(list *models.TaskList) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// Check if list exists
	listDir := filepath.Join(fs.baseDir, "lists", list.ID)
	if _, err := os.Stat(listDir); os.IsNotExist(err) {
		return fmt.Errorf("list not found: %s", list.ID)
	}

	// Update timestamp
	list.UpdatedAt = time.Now()

	// Write list file
	listPath := filepath.Join(listDir, "list.json")
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize list: %w", err)
	}

	if err := os.WriteFile(listPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write list file: %w", err)
	}

	return nil
}

// DeleteList deletes a task list and all its tasks
func (fs *FileStore) DeleteList(id string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	listDir := filepath.Join(fs.baseDir, "lists", id)
	if _, err := os.Stat(listDir); os.IsNotExist(err) {
		return fmt.Errorf("list not found: %s", id)
	}

	if err := os.RemoveAll(listDir); err != nil {
		return fmt.Errorf("failed to delete list: %w", err)
	}

	return nil
}

// Task Methods

// GetAllTasks returns all tasks across all lists
func (fs *FileStore) GetAllTasks() ([]models.Task, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	var allTasks []models.Task

	// Get all lists
	lists, err := fs.GetAllLists()
	if err != nil {
		return nil, err
	}

	// For each list, get all tasks
	for _, list := range lists {
		tasks, err := fs.GetTasksForList(list.ID)
		if err != nil {
			// Skip if tasks cannot be read
			continue
		}
		allTasks = append(allTasks, tasks...)
	}

	return allTasks, nil
}

// GetTasksByList returns all tasks for a specific list
func (fs *FileStore) GetTasksByList(listID string) ([]models.Task, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	return fs.GetTasksForList(listID)
}

// GetTasksForList gets tasks for a list
func (fs *FileStore) GetTasksForList(listID string) ([]models.Task, error) {
	tasksDir := filepath.Join(fs.baseDir, "lists", listID, "tasks")
	
	// Check if tasks directory exists
	if _, err := os.Stat(tasksDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("list not found: %s", listID)
	}

	files, err := os.ReadDir(tasksDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks directory: %w", err)
	}

	var tasks []models.Task
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			taskPath := filepath.Join(tasksDir, file.Name())
			
			// Read task file
			data, err := os.ReadFile(taskPath)
			if err != nil {
				// Skip if task file cannot be read
				continue
			}

			var task models.Task
			if err := json.Unmarshal(data, &task); err != nil {
				// Skip if task file cannot be parsed
				continue
			}

			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// GetTask returns a single task by ID
func (fs *FileStore) GetTask(listID, taskID string) (*models.Task, error) {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	// Check if the specific list's task exists first
	taskPath := filepath.Join(fs.baseDir, "lists", listID, "tasks", taskID+".json")
	_, err := os.Stat(taskPath)
	if err == nil {
		// Found the task, read it
		data, err := os.ReadFile(taskPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read task: %w", err)
		}

		var task models.Task
		if err := json.Unmarshal(data, &task); err != nil {
			return nil, fmt.Errorf("failed to parse task: %w", err)
		}

		return &task, nil
	}

	// If we couldn't find it in the specific list, search all lists
	if listID != "" {
		// Get all lists
		lists, err := fs.GetAllLists()
		if err != nil {
			return nil, err
		}

		// Search for the task in all lists
		for _, list := range lists {
			taskPath := filepath.Join(fs.baseDir, "lists", list.ID, "tasks", taskID+".json")
			_, err := os.Stat(taskPath)
			if err == nil {
				// Found the task, read it
				data, err := os.ReadFile(taskPath)
				if err != nil {
					return nil, fmt.Errorf("failed to read task: %w", err)
				}

				var task models.Task
				if err := json.Unmarshal(data, &task); err != nil {
					return nil, fmt.Errorf("failed to parse task: %w", err)
				}

				return &task, nil
			}
		}
	}

	return nil, fmt.Errorf("task not found: %s", taskID)
}

// CreateTask creates a new task
func (fs *FileStore) CreateTask(task *models.Task) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// Check if list exists
	listDir := filepath.Join(fs.baseDir, "lists", task.ListID)
	if _, err := os.Stat(listDir); os.IsNotExist(err) {
		return fmt.Errorf("list not found: %s", task.ListID)
	}

	// Create tasks directory if it doesn't exist
	tasksDir := filepath.Join(listDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return fmt.Errorf("failed to create tasks directory: %w", err)
	}

	// Set timestamps
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now
	if task.StateTime.IsZero() {
		task.StateTime = now
	}

	// Write task file
	taskPath := filepath.Join(tasksDir, task.ID+".json")
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize task: %w", err)
	}

	if err := os.WriteFile(taskPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write task file: %w", err)
	}

	return nil
}

// UpdateTask updates an existing task
func (fs *FileStore) UpdateTask(task *models.Task) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// Ensure list directory exists
	listDir := filepath.Join(fs.baseDir, "lists", task.ListID)
	if _, err := os.Stat(listDir); os.IsNotExist(err) {
		return fmt.Errorf("list directory not found: %s", task.ListID)
	}

	// Ensure tasks directory exists
	tasksDir := filepath.Join(listDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0755); err != nil {
		return fmt.Errorf("failed to create tasks directory: %w", err)
	}
	
	// Set the task path
	taskPath := filepath.Join(tasksDir, task.ID+".json")

	// Update timestamp
	task.UpdatedAt = time.Now()

	// Write task file
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize task: %w", err)
	}

	if err := os.WriteFile(taskPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write task file: %w", err)
	}

	return nil
}

// MoveTask moves a task from one list to another
func (fs *FileStore) MoveTask(originalListID, taskID, newListID string) (*models.Task, error) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	
	// Ensure the original list's tasks directory exists
	originalTasksDir := filepath.Join(fs.baseDir, "lists", originalListID, "tasks")
	if err := os.MkdirAll(originalTasksDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to ensure original tasks directory: %w", err)
	}
	
	// Get the task
	originalTaskPath := filepath.Join(originalTasksDir, taskID+".json")
	data, err := os.ReadFile(originalTaskPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("task not found: %s/%s", originalListID, taskID)
		}
		return nil, fmt.Errorf("failed to read task: %w", err)
	}
	
	var task models.Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to parse task: %w", err)
	}
	
	// Update the list ID
	task.ListID = newListID
	task.UpdatedAt = time.Now()
	
	// Check if the destination list exists
	newListDir := filepath.Join(fs.baseDir, "lists", newListID)
	if _, err := os.Stat(newListDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("destination list not found: %s", newListID)
	}
	
	// Ensure the new list's tasks directory exists
	newTasksDir := filepath.Join(newListDir, "tasks")
	if err := os.MkdirAll(newTasksDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination tasks directory: %w", err)
	}
	
	// Write the task to the new list
	newTaskPath := filepath.Join(newTasksDir, taskID+".json")
	
	data, err = json.MarshalIndent(task, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize task: %w", err)
	}
	
	if err := os.WriteFile(newTaskPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write task file: %w", err)
	}
	
	// Delete the task from the original list
	if err := os.Remove(originalTaskPath); err != nil {
		return nil, fmt.Errorf("failed to delete original task: %w", err)
	}
	
	return &task, nil
}

// DeleteTask deletes a task
func (fs *FileStore) DeleteTask(listID, taskID string) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	// First try to delete from the specific list
	taskPath := filepath.Join(fs.baseDir, "lists", listID, "tasks", taskID+".json")
	_, err := os.Stat(taskPath)
	if err == nil {
		// Found the task, delete it
		if err := os.Remove(taskPath); err != nil {
			return fmt.Errorf("failed to delete task: %w", err)
		}
		return nil
	}

	// If not found in the specific list, search all lists
	lists, err := fs.GetAllLists()
	if err != nil {
		return err
	}

	// Search for the task in all lists
	for _, list := range lists {
		taskPath := filepath.Join(fs.baseDir, "lists", list.ID, "tasks", taskID+".json")
		_, err := os.Stat(taskPath)
		if err == nil {
			// Found the task, delete it
			if err := os.Remove(taskPath); err != nil {
				return fmt.Errorf("failed to delete task: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("task not found: %s", taskID)
}