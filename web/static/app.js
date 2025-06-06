// Task management app JavaScript enhancements

// Clear forms after successful submission
document.addEventListener('htmx:afterRequest', function(event) {
    if (event.detail.successful && event.target.tagName === 'FORM') {
        event.target.reset();
    }
});

// List filter functionality
document.addEventListener('DOMContentLoaded', function() {
    const filterForm = document.getElementById('list-filter-form');
    if (filterForm) {
        // Set up filter event handlers
        const allCheckbox = filterForm.querySelector('[data-filter-all]');
        const listCheckboxes = filterForm.querySelectorAll('[data-list-id]');
        
        // Function to update task visibility based on selected filters
        function updateTaskVisibility() {
            // Get all tasks (both regular and kanban tasks)
            const tasks = document.querySelectorAll('.task, .kanban-task');
            const selectedListIds = [];
            let showAll = allCheckbox.checked;
            
            // If no specific lists are selected or "All Lists" is checked, show all tasks
            if (showAll) {
                tasks.forEach(task => task.style.display = '');
                return;
            }
            
            // Collect selected list IDs
            listCheckboxes.forEach(checkbox => {
                if (checkbox.checked) {
                    selectedListIds.push(checkbox.getAttribute('data-list-id'));
                }
            });
            
            // If no lists are selected after unchecking "All Lists", show nothing
            if (selectedListIds.length === 0) {
                tasks.forEach(task => task.style.display = 'none');
                return;
            }
            
            // Show only tasks from selected lists
            tasks.forEach(task => {
                const listId = task.getAttribute('data-list-id');
                task.style.display = selectedListIds.includes(listId) ? '' : 'none';
            });
        }
        
        // Add event listeners
        allCheckbox.addEventListener('change', function() {
            if (this.checked) {
                // When "All Lists" is checked, uncheck all individual lists
                listCheckboxes.forEach(checkbox => checkbox.checked = false);
            }
            updateTaskVisibility();
        });
        
        listCheckboxes.forEach(checkbox => {
            checkbox.addEventListener('change', function() {
                // If any individual list is checked, uncheck "All Lists"
                if (this.checked) {
                    allCheckbox.checked = false;
                }
                
                // If no individual lists are checked, check "All Lists"
                let anyChecked = false;
                listCheckboxes.forEach(cb => {
                    if (cb.checked) anyChecked = true;
                });
                
                if (!anyChecked) {
                    allCheckbox.checked = true;
                }
                
                updateTaskVisibility();
            });
        });
    }
});

// Handle custom clear form event
document.addEventListener('clearListForm', function(event) {
    const form = document.querySelector('.new-list-form form');
    if (form) {
        form.reset();
    }
});

// Handle custom clear task form event
document.addEventListener('clearTaskForm', function(event) {
    const form = document.querySelector('.new-task-form form');
    if (form) {
        form.reset();
    }
});

// Modal functionality
document.addEventListener('DOMContentLoaded', function() {
    // Modal handling
    const modalBackdrop = document.createElement('div');
    modalBackdrop.className = 'modal-backdrop';
    document.body.appendChild(modalBackdrop);

    // Close modal when clicking on backdrop
    modalBackdrop.addEventListener('click', function(event) {
        if (event.target === modalBackdrop) {
            closeModal();
        }
    });

    // Handle edit task clicks
    document.addEventListener('click', function(event) {
        // Check if clicked on a task item but not on its buttons
        const taskItem = event.target.closest('.task-state-todo, .task-state-in_progress, .task-state-blocked, .task-state-done, .task-state-todo, .kanban-task, .task');
        if (taskItem && !event.target.closest('button, select, a, .task-actions')) {
            const listId = taskItem.getAttribute('data-list-id');
            const taskId = taskItem.getAttribute('data-task-id');
            
            if (listId && taskId) {
                openEditTaskModal(listId, taskId);
            } else {
                console.error('Missing task data attributes', { element: taskItem });
            }
        }
    });

    // Function to load task data and open the edit modal
    function openEditTaskModal(listId, taskId) {
        
        fetch(`/api/tasks/${listId}/${taskId}`)
            .then(response => {
                if (!response.ok) {
                    throw new Error(`Failed to fetch task: ${response.status} ${response.statusText}`);
                }
                return response.json();
            })
            .then(task => {
                // Validate the task data
                if (!task || !task.id) {
                    throw new Error('Invalid task data received from API');
                }
                
                // Get all lists for the list selector
                return fetch('/api/lists')
                    .then(response => {
                        if (!response.ok) {
                            throw new Error(`Failed to fetch lists: ${response.status} ${response.statusText}`);
                        }
                        return response.json();
                    })
                    .then(lists => {
                        if (!Array.isArray(lists)) {
                            console.warn('Lists data is not an array:', lists);
                            lists = [];
                        }
                        
                        // Ensure task has the correct ID attributes even if API didn't return them
                        if (!task.id) task.id = taskId;
                        if (!task.list_id) task.list_id = listId;
                        
                        showTaskEditModal(task, lists);
                    });
            })
            .catch(error => {
                console.error('Error in task edit flow:', error);
                alert(`Could not edit task: ${error.message}`);
                closeModal();
            });
    }

    // Function to display the task edit modal
    function showTaskEditModal(task, lists) {
        // Validate task data
        if (!task || !task.id || !task.list_id) {
            console.error('Invalid task data received', task);
            alert('Cannot edit task: Invalid task data');
            closeModal();
            return;
        }

        // Format date for input field if present
        const formattedDate = task.due_date ? new Date(task.due_date).toISOString().split('T')[0] : '';
        
        // Get target selector based on current view
        const targetSelector = window.location.pathname.includes('/kanban/') ? '.kanban-board' : '.tasks-container';
        
        // Create modal structure
        modalBackdrop.innerHTML = '';
        const modalTemplate = `
            <div class="modal-content">
                <div class="modal-header">
                    <h3>Edit Task</h3>
                    <button class="modal-close" onclick="closeModal()">×</button>
                </div>
                <div class="modal-body">
                    <form id="edit-task-form" data-task-id="${task.id}" data-list-id="${task.list_id}" data-target="${targetSelector}" enctype="application/x-www-form-urlencoded">
                        <input type="hidden" id="edit-id" name="id" value="${task.id}">
                        <input type="hidden" id="edit-original-list-id" name="original_list_id" value="${task.list_id}">
                        
                        <div>
                            <label for="edit-title">Title:</label>
                            <input type="text" id="edit-title" name="title" value="${task.title || ''}" required placeholder="Task title (required)">
                        </div>
                        
                        <div>
                            <label for="edit-description">Description:</label>
                            <textarea id="edit-description" name="description" rows="3">${task.description || ''}</textarea>
                        </div>
                        
                        <div>
                            <label for="edit-state">State:</label>
                            <select id="edit-state" name="state">
                                ${['todo', 'in_progress', 'blocked', 'done'].map(state => 
                                    `<option value="${state}" ${task.state === state ? 'selected' : ''}>
                                        ${state === 'todo' ? 'Todo' : 
                                          state === 'in_progress' ? 'In Progress' : 
                                          state === 'blocked' ? 'Blocked' : 'Done'}
                                    </option>`
                                ).join('')}
                            </select>
                        </div>
                        
                        <div>
                            <label for="edit-list">Task List:</label>
                            <select id="edit-list" name="list_id">
                                ${lists.map(list => 
                                    `<option value="${list.id}" ${list.id === task.list_id ? 'selected' : ''}>
                                        ${list.name}
                                    </option>`
                                ).join('')}
                            </select>
                        </div>
                        
                        <div>
                            <label for="edit-due-date">Due Date:</label>
                            <input type="date" id="edit-due-date" name="due_date" value="${formattedDate}">
                        </div>
                    </form>
                </div>
                <div class="modal-footer">
                    <button type="button" class="button" onclick="closeModal()">Cancel</button>
                    <button type="button" class="button" data-task-id="${task.id}" data-list-id="${task.list_id}" 
                            onclick="submitTaskEdit(event, '${task.id}', '${task.list_id}')">
                        Save Changes
                    </button>
                </div>
            </div>
        `;
        
        // Add to page and show
        modalBackdrop.innerHTML = modalTemplate;
        modalBackdrop.classList.add('show');
        
        // Ensure description is set correctly (textarea quirk) and hide other modals
        setTimeout(() => {
            // Fix for textareas - they need value instead of textContent
            const descField = document.getElementById('edit-description');
            if (descField) {
                descField.value = task.description || '';
            }
            
            // Hide any static modals in the page
            document.querySelectorAll('.modal').forEach(modal => {
                if (modal !== modalBackdrop) {
                    modal.style.display = 'none';
                }
            });
        }, 50);
    }

    // Make functions globally available
    window.closeModal = function() {
        modalBackdrop.classList.remove('show');
        modalBackdrop.innerHTML = '';
        
        // Ensure static modals remain hidden
        const staticModals = document.querySelectorAll('.modal');
        staticModals.forEach(modal => {
            if (modal !== modalBackdrop) {
                modal.style.display = 'none';
            }
        });
    };
    
    window.submitTaskEdit = function(event, taskIdParam, listIdParam) {
        // Prevent any default action
        if (event) event.preventDefault();
        
        try {
            // Find the form directly from the modal backdrop
            const form = document.querySelector('.modal-backdrop #edit-task-form');
            if (!form) {
                alert("Error: Form not found");
                return;
            }

            // Get IDs from various sources, with form being primary now
            const taskId = form.elements.id?.value || 
                          taskIdParam || 
                          form.getAttribute('data-task-id');
                          
            const originalListId = listIdParam || 
                                 form.getAttribute('data-list-id');
                                 
            // Get the potentially new list ID from the form
            const newListId = form.elements.list_id?.value;

            // Validate IDs
            if (!taskId || !originalListId) {
                alert("Error: Could not find task or list ID");
                return;
            }
            
            // Use FormData to automatically collect all form values
            const formData = new FormData(form);
            
            // Convert FormData to URLSearchParams for sending
            const urlParams = new URLSearchParams();
            
            for (const [key, value] of formData.entries()) {
                urlParams.append(key, value);
            }
            
            // Ensure list_id is included
            if (!formData.has('list_id') && newListId) {
                urlParams.append('list_id', newListId);
            }
            
            // Make the API request - always use the original list ID in the URL
            const apiUrl = `/api/tasks/${originalListId}/${taskId}`;
            
            fetch(apiUrl, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded'
                },
                body: urlParams.toString()
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP error ${response.status}`);
                }
                return response.text().then(text => {
                    // Try to parse as JSON, but handle empty or non-JSON responses
                    try {
                        return text ? JSON.parse(text) : {};
                    } catch (e) {
                        return { html: text };
                    }
                });
            })
            .then(data => {
                closeModal();
                
                // Check if we're in the all-kanban view
                const isAllKanban = window.location.pathname.includes('/all-kanban');
                
                if (isAllKanban || (originalListId !== newListId)) {
                    // For all-kanban view or when the list changed, reload the page
                    window.location.reload();
                }
            })
            .catch(error => {
                console.error("Error submitting task:", error);
                alert(`Failed to update task: ${error.message}`);
                closeModal();
            });
        } catch (error) {
            console.error("Error in submitTaskEdit:", error);
            alert(`An error occurred: ${error.message}`);
        }
    };
});