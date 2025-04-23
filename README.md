# Task Management System

A simple task management system with a Go backend API and HTMX frontend.

## Features

- JSON REST API with full CRUD support
- Support for multiple task lists
- Task states (Todo, In Progress, Blocked, Done)
- Subtasks support
- Task notes
- Due dates
- State duration tracking
- Export to markdown
- Flat file storage

## Views

- All tasks view across all lists
- List-specific view
- Kanban board view

## Installation

### Requirements

- Go 1.22 or higher

### Setup

1. Clone the repository:
```bash
git clone https://github.com/jbutlerdev/tasks.git
cd tasks
```

2. Download dependencies:
```bash
go mod download
```

3. Build the application:
```bash
go build -o tasks
```

## Usage

### Running the server

```bash
./tasks --port 8080 --data ./data
```

Options:
- `--port`: Port to run the server on (default: 8080)
- `--data`: Directory to store task data (default: ./data)

### API Endpoints

#### Task Lists

- `GET /api/lists`: Get all task lists
- `POST /api/lists`: Create a new task list
- `GET /api/lists/{listID}`: Get a specific task list
- `PUT /api/lists/{listID}`: Update a task list
- `DELETE /api/lists/{listID}`: Delete a task list
- `GET /api/lists/{listID}/tasks`: Get all tasks for a list
- `POST /api/lists/{listID}/tasks`: Create a new task in a list

#### Tasks

- `GET /api/tasks`: Get all tasks across all lists
- `GET /api/tasks/{listID}/{taskID}`: Get a specific task
- `PUT /api/tasks/{listID}/{taskID}`: Update a task
- `DELETE /api/tasks/{listID}/{taskID}`: Delete a task

#### Export

- `GET /api/export`: Export all tasks as markdown

### Web UI

- `/`: View all tasks across all lists
- `/lists`: View all task lists
- `/lists/{listID}`: View tasks for a specific list
- `/kanban/{listID}`: View tasks for a list in kanban board format

## Data Storage

Task data is stored in flat JSON files, organized by task list:

```
data/
└── lists/
    ├── list-id-1/
    │   ├── list.json
    │   └── tasks/
    │       ├── task-id-1.json
    │       ├── task-id-2.json
    │       └── ...
    └── list-id-2/
        ├── list.json
        └── tasks/
            ├── task-id-3.json
            └── ...
```

## License

MIT