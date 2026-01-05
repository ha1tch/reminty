import React, { useState } from 'react';

function TaskCard({ task }) {
  const statusClass = task.completed ? 'completed' : 'pending';
  return (
    <div className={`task-card ${statusClass}`}>
      <div className="task-header">
        <input
          type="checkbox"
          checked={task.completed}
          aria-label="Toggle task"
        />
        <h3>{task.title}</h3>
        <span className={`priority priority-${task.priority}`}>
          {task.priority}
        </span>
      </div>
      {task.description && (
        <p className="task-description">{task.description}</p>
      )}
      <div className="task-footer">
        <span className="task-date">Due: {task.dueDate}</span>
        <button className="btn-delete">Delete</button>
      </div>
    </div>
  );
}

function TaskBoard({ initialTasks }) {
  const [tasks, setTasks] = useState(initialTasks);
  const [filter, setFilter] = useState('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [sortBy, setSortBy] = useState('date');

  const activeTasks = tasks.filter(task => !task.completed);
  const completedTasks = tasks.filter(task => task.completed);
  
  const filteredTasks = tasks.filter(task => 
    task.title.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className="task-board">
      <header className="board-header">
        <h1>Task Board</h1>
        <div className="stats">
          <span className="stat">{tasks.length} total</span>
          <span className="stat">{completedTasks.length} done</span>
          <span className="stat">{activeTasks.length} remaining</span>
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
            aria-selected={filter === 'all'}
            onClick={() => setFilter('all')}
          >
            All
          </button>
          <button
            role="tab"
            aria-selected={filter === 'active'}
            onClick={() => setFilter('active')}
          >
            Active
          </button>
          <button
            role="tab"
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
        {filteredTasks.length > 0 ? (
          <div className="tasks">
            {filteredTasks.map(task => (
              <TaskCard key={task.id} task={task} />
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

      {completedTasks.length > 0 && (
        <footer className="board-footer">
          <button 
            className="btn-clear"
            onClick={() => setTasks(activeTasks)}
          >
            Clear completed
          </button>
        </footer>
      )}
    </div>
  );
}

export default TaskBoard;
