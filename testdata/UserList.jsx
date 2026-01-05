import React, { useState, useEffect } from 'react';

function UserCard({ name, email, avatar, isActive }) {
  return (
    <div className="user-card">
      <img src={avatar} alt={name} className="avatar" />
      <div className="user-info">
        <h3>{name}</h3>
        <p className="email">{email}</p>
        {isActive && <span className="badge">Active</span>}
      </div>
    </div>
  );
}

function UserList({ users }) {
  const [filter, setFilter] = useState('');
  const [activeTab, setActiveTab] = useState('all');

  const filteredUsers = users.filter(user => 
    user.name.toLowerCase().includes(filter.toLowerCase())
  );

  return (
    <div className="user-list">
      <div className="tabs" role="tablist">
        <button 
          role="tab"
          aria-selected={activeTab === 'all'}
          onClick={() => setActiveTab('all')}
        >
          All Users
        </button>
        <button 
          role="tab"
          aria-selected={activeTab === 'active'}
          onClick={() => setActiveTab('active')}
        >
          Active Only
        </button>
      </div>

      <input
        type="search"
        placeholder="Search users..."
        value={filter}
        onChange={(e) => setFilter(e.target.value)}
      />

      <div className="results">
        {filteredUsers.length > 0 ? (
          <ul>
            {filteredUsers.map((user, index) => (
              <li key={user.id}>
                <UserCard
                  name={user.name}
                  email={user.email}
                  avatar={user.avatar}
                  isActive={user.active}
                />
              </li>
            ))}
          </ul>
        ) : (
          <p className="empty">No users found</p>
        )}
      </div>
    </div>
  );
}

export default UserList;
