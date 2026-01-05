import React, { useState } from 'react';

function TaskCard({ task, onToggle, onDelete }) {
  return (
    <div className={`task-card ${task.completed ? 'completed' : ''}`}>
      <div className="task-header">
        <input
          type="checkbox"
          checked={task.completed}
          onChange={() => onToggle(task.id)}
          aria-label={`Mark "${task.title}" as ${task.completed ? 'incomplete' : 'complete'}`}
        />
        <h3 className={task.completed ? 'line-through' : ''}>{task.title}</h3>
        <span className={`priority priority-${task.priority}`}>
          {task.priority}
        </span>
      </div>
      {task.description && (
        <p className="task-description">{task.description}</p>
      )}
      <div className="task-footer">
        <span className="task-date">Due: {task.dueDate}</span>
        <button 
          className="btn-delete"
          onClick={() => onDelete(task.id)}
          aria-label={`Delete task: ${task.title}`}
        >
          Delete
        </button>
      </div>
    </div>
  );
}

function TaskBoard({ initialTasks }) {
  const [tasks, setTasks] = useState(initialTasks);
  const [filter, setFilter] = useState('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [sortBy, setSortBy] = useState('date');

  // Derived state
  const filteredTasks = tasks
    .filter(task => {
      if (filter === 'active') return !task.completed;
      if (filter === 'completed') return task.completed;
      return true;
    })
    .filter(task => 
      task.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      task.description?.toLowerCase().includes(searchQuery.toLowerCase())
    );

  const sortedTasks = filteredTasks.sort((a, b) => {
    if (sortBy === 'priority') {
      const priorityOrder = { high: 0, medium: 1, low: 2 };
      return priorityOrder[a.priority] - priorityOrder[b.priority];
    }
    return new Date(a.dueDate) - new Date(b.dueDate);
  });

  const stats = {
    total: tasks.length,
    completed: tasks.filter(t => t.completed).length,
    active: tasks.filter(t => !t.completed).length
  };

  const handleToggle = (id) => {
    setTasks(tasks.map(task =>
      task.id === id ? { ...task, completed: !task.completed } : task
    ));
  };

  const handleDelete = (id) => {
    setTasks(tasks.filter(task => task.id !== id));
  };

  const handleClearCompleted = () => {
    setTasks(tasks.filter(task => !task.completed));
  };

  return (
    <div className="task-board">
      <header className="board-header">
        <h1>Task Board</h1>
        <div className="stats">
          <span className="stat">{stats.total} total</span>
          <span className="stat completed">{stats.completed} done</span>
          <span className="stat active">{stats.active} remaining</span>
        </div>
      </header>

      <div className="controls">
        <input
          type="search"
          className="search-input"
          placeholder="Search tasks..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
        />

        <div className="filter-tabs" role="tablist">
          <button
            role="tab"
            className={filter === 'all' ? 'active' : ''}
            aria-selected={filter === 'all'}
            onClick={() => setFilter('all')}
          >
            All
          </button>
          <button
            role="tab"
            className={filter === 'active' ? 'active' : ''}
            aria-selected={filter === 'active'}
            onClick={() => setFilter('active')}
          >
            Active
          </button>
          <button
            role="tab"
            className={filter === 'completed' ? 'active' : ''}
            aria-selected={filter === 'completed'}
            onClick={() => setFilter('completed')}
          >
            Completed
          </button>
        </div>

        <select 
          className="sort-select"
          value={sortBy}
          onChange={(e) => setSortBy(e.target.value)}
        >
          <option value="date">Sort by Date</option>
          <option value="priority">Sort by Priority</option>
        </select>
      </div>

      <main className="task-list">
        {sortedTasks.length > 0 ? (
          <div className="tasks">
            {sortedTasks.map(task => (
              <TaskCard
                key={task.id}
                task={task}
                onToggle={handleToggle}
                onDelete={handleDelete}
              />
            ))}
          </div>
        ) : (
          <div className="empty-state">
            <p>No tasks found</p>
            {filter !== 'all' && (
              <button onClick={() => setFilter('all')}>Show all tasks</button>
            )}
          </div>
        )}
      </main>

      {stats.completed > 0 && (
        <footer className="board-footer">
          <button 
            className="btn-clear"
            onClick={handleClearCompleted}
          >
            Clear {stats.completed} completed task{stats.completed !== 1 ? 's' : ''}
          </button>
        </footer>
      )}
    </div>
  );
}

export default TaskBoard;
