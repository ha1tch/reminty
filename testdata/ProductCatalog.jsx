import React, { useState } from 'react';

function StarRating({ rating, maxStars }) {
  const stars = [];
  for (let i = 1; i <= maxStars; i++) {
    stars.push(i <= rating ? 'filled' : 'empty');
  }
  return (
    <div className="star-rating" aria-label={`${rating} out of ${maxStars} stars`}>
      {stars.map((type, index) => (
        <span key={index} className={`star star-${type}`}>★</span>
      ))}
    </div>
  );
}

function ProductCard({ product, onAddToCart }) {
  const isOnSale = product.salePrice && product.salePrice < product.price;
  const displayPrice = isOnSale ? product.salePrice : product.price;
  const savings = isOnSale ? product.price - product.salePrice : 0;

  return (
    <article className="product-card">
      <div className="product-image">
        <img src={product.imageUrl} alt={product.name} />
        {isOnSale && <span className="sale-badge">Sale!</span>}
        {product.inventory < 5 && product.inventory > 0 && (
          <span className="low-stock">Only {product.inventory} left</span>
        )}
      </div>
      
      <div className="product-info">
        <h3 className="product-name">{product.name}</h3>
        <p className="product-category">{product.category}</p>
        
        <StarRating rating={product.rating} maxStars={5} />
        <span className="review-count">({product.reviewCount} reviews)</span>
        
        <div className="product-pricing">
          {isOnSale && (
            <span className="original-price">${product.price}</span>
          )}
          <span className="current-price">${displayPrice}</span>
          {savings > 0 && (
            <span className="savings">Save ${savings}</span>
          )}
        </div>
        
        {product.inventory > 0 ? (
          <button 
            className="btn-add-cart"
            onClick={() => onAddToCart(product)}
          >
            Add to Cart
          </button>
        ) : (
          <button className="btn-out-of-stock" disabled>
            Out of Stock
          </button>
        )}
      </div>
    </article>
  );
}

function CartPreview({ items, total }) {
  const itemCount = items.length;
  
  return (
    <div className="cart-preview">
      <div className="cart-icon">
        <span className="cart-badge">{itemCount}</span>
      </div>
      {itemCount > 0 && (
        <div className="cart-dropdown">
          <ul className="cart-items">
            {items.map(item => (
              <li key={item.id} className="cart-item">
                <span className="item-name">{item.name}</span>
                <span className="item-price">${item.price}</span>
              </li>
            ))}
          </ul>
          <div className="cart-total">
            <strong>Total: ${total}</strong>
          </div>
          <button className="btn-checkout">Checkout</button>
        </div>
      )}
    </div>
  );
}

function ProductCatalog({ products }) {
  const [category, setCategory] = useState('all');
  const [sortBy, setSortBy] = useState('name');
  const [priceRange, setPriceRange] = useState('all');
  const [cartItems, setCartItems] = useState([]);

  const categories = ['all', 'electronics', 'clothing', 'home', 'sports'];

  const filteredProducts = products.filter(p => {
    if (category !== 'all' && p.category !== category) return false;
    if (priceRange === 'under50' && p.price >= 50) return false;
    if (priceRange === '50to100' && (p.price < 50 || p.price > 100)) return false;
    if (priceRange === 'over100' && p.price <= 100) return false;
    return true;
  });

  const sortedProducts = filteredProducts.sort((a, b) => {
    if (sortBy === 'price-low') return a.price - b.price;
    if (sortBy === 'price-high') return b.price - a.price;
    if (sortBy === 'rating') return b.rating - a.rating;
    return a.name.localeCompare(b.name);
  });

  const cartTotal = cartItems.reduce((sum, item) => sum + item.price, 0);

  const handleAddToCart = (product) => {
    setCartItems([...cartItems, product]);
  };

  return (
    <div className="product-catalog">
      <header className="catalog-header">
        <h1>Product Catalog</h1>
        <CartPreview items={cartItems} total={cartTotal} />
      </header>

      <aside className="filters">
        <div className="filter-group">
          <label htmlFor="category">Category</label>
          <select 
            id="category"
            value={category}
            onChange={(e) => setCategory(e.target.value)}
          >
            {categories.map(cat => (
              <option key={cat} value={cat}>
                {cat === 'all' ? 'All Categories' : cat}
              </option>
            ))}
          </select>
        </div>

        <div className="filter-group">
          <label htmlFor="price">Price Range</label>
          <select
            id="price"
            value={priceRange}
            onChange={(e) => setPriceRange(e.target.value)}
          >
            <option value="all">Any Price</option>
            <option value="under50">Under $50</option>
            <option value="50to100">$50 - $100</option>
            <option value="over100">Over $100</option>
          </select>
        </div>

        <div className="filter-group">
          <label htmlFor="sort">Sort By</label>
          <select
            id="sort"
            value={sortBy}
            onChange={(e) => setSortBy(e.target.value)}
          >
            <option value="name">Name</option>
            <option value="price-low">Price: Low to High</option>
            <option value="price-high">Price: High to Low</option>
            <option value="rating">Top Rated</option>
          </select>
        </div>
      </aside>

      <main className="product-grid">
        <p className="results-count">
          Showing {sortedProducts.length} of {products.length} products
        </p>
        
        {sortedProducts.length > 0 ? (
          <div className="products">
            {sortedProducts.map(product => (
              <ProductCard 
                key={product.id}
                product={product}
                onAddToCart={handleAddToCart}
              />
            ))}
          </div>
        ) : (
          <div className="no-results">
            <p>No products match your filters</p>
            <button onClick={() => {
              setCategory('all');
              setPriceRange('all');
            }}>
              Clear Filters
            </button>
          </div>
        )}
      </main>
    </div>
  );
}

export default ProductCatalog;
