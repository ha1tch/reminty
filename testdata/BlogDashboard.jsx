import React, { useState } from 'react';

function PostCard({ post }) {
  const isPublished = post.status === 'published';
  const isDraft = post.status === 'draft';
  
  return (
    <article className={`post-card ${post.status}`}>
      <header className="post-header">
        <h2 className="post-title">{post.title}</h2>
        <div className="post-meta">
          <span className="author">By {post.author}</span>
          <time className="date">{post.date}</time>
          {post.category && (
            <span className="category">{post.category}</span>
          )}
        </div>
      </header>
      
      <p className="post-excerpt">{post.excerpt}</p>
      
      <footer className="post-footer">
        <div className="post-stats">
          <span className="views">{post.views} views</span>
          <span className="comments">{post.comments} comments</span>
          {post.likes > 0 && (
            <span className="likes">{post.likes} likes</span>
          )}
        </div>
        
        <div className="post-actions">
          {isPublished && (
            <button className="btn btn-view">View</button>
          )}
          <button className="btn btn-edit">Edit</button>
          {isDraft && (
            <button className="btn btn-publish">Publish</button>
          )}
        </div>
      </footer>
    </article>
  );
}

function StatsWidget({ posts }) {
  const totalPosts = posts.length;
  const publishedCount = posts.filter(p => p.status === 'published').length;
  const draftCount = posts.filter(p => p.status === 'draft').length;
  const totalViews = posts.reduce((sum, p) => sum + p.views, 0);
  
  return (
    <aside className="stats-widget">
      <h3>Dashboard Stats</h3>
      <ul className="stats-list">
        <li className="stat-item">
          <span className="stat-label">Total Posts</span>
          <span className="stat-value">{totalPosts}</span>
        </li>
        <li className="stat-item">
          <span className="stat-label">Published</span>
          <span className="stat-value">{publishedCount}</span>
        </li>
        <li className="stat-item">
          <span className="stat-label">Drafts</span>
          <span className="stat-value">{draftCount}</span>
        </li>
        <li className="stat-item">
          <span className="stat-label">Total Views</span>
          <span className="stat-value">{totalViews}</span>
        </li>
      </ul>
    </aside>
  );
}

function BlogDashboard({ initialPosts }) {
  const [posts, setPosts] = useState(initialPosts);
  const [statusFilter, setStatusFilter] = useState('all');
  const [searchTerm, setSearchTerm] = useState('');
  const [sortOrder, setSortOrder] = useState('newest');

  const filteredPosts = posts.filter(post => {
    if (statusFilter !== 'all' && post.status !== statusFilter) {
      return false;
    }
    if (searchTerm && !post.title.toLowerCase().includes(searchTerm.toLowerCase())) {
      return false;
    }
    return true;
  });

  const sortedPosts = filteredPosts.sort((a, b) => {
    if (sortOrder === 'newest') return new Date(b.date) - new Date(a.date);
    if (sortOrder === 'oldest') return new Date(a.date) - new Date(b.date);
    if (sortOrder === 'popular') return b.views - a.views;
    return a.title.localeCompare(b.title);
  });

  return (
    <div className="blog-dashboard">
      <header className="dashboard-header">
        <h1>Blog Dashboard</h1>
        <button 
          className="btn btn-primary"
          onClick={() => console.log('New post')}
        >
          New Post
        </button>
      </header>

      <div className="dashboard-content">
        <StatsWidget posts={posts} />

        <main className="posts-section">
          <div className="posts-toolbar">
            <input
              type="search"
              className="search-input"
              placeholder="Search posts..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />

            <div className="filter-group" role="tablist">
              <button
                role="tab"
                className={statusFilter === 'all' ? 'active' : ''}
                aria-selected={statusFilter === 'all'}
                onClick={() => setStatusFilter('all')}
              >
                All Posts
              </button>
              <button
                role="tab"
                className={statusFilter === 'published' ? 'active' : ''}
                aria-selected={statusFilter === 'published'}
                onClick={() => setStatusFilter('published')}
              >
                Published
              </button>
              <button
                role="tab"
                className={statusFilter === 'draft' ? 'active' : ''}
                aria-selected={statusFilter === 'draft'}
                onClick={() => setStatusFilter('draft')}
              >
                Drafts
              </button>
            </div>

            <select
              className="sort-select"
              value={sortOrder}
              onChange={(e) => setSortOrder(e.target.value)}
            >
              <option value="newest">Newest First</option>
              <option value="oldest">Oldest First</option>
              <option value="popular">Most Popular</option>
              <option value="title">By Title</option>
            </select>
          </div>

          <div className="posts-list">
            {sortedPosts.length > 0 ? (
              sortedPosts.map(post => (
                <PostCard key={post.id} post={post} />
              ))
            ) : (
              <div className="empty-state">
                <p>No posts found matching your criteria.</p>
                {statusFilter !== 'all' && (
                  <button 
                    className="btn btn-link"
                    onClick={() => setStatusFilter('all')}
                  >
                    Show all posts
                  </button>
                )}
              </div>
            )}
          </div>

          {sortedPosts.length > 0 && (
            <p className="results-summary">
              Showing {sortedPosts.length} of {posts.length} posts
            </p>
          )}
        </main>
      </div>
    </div>
  );
}

export default BlogDashboard;
