CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clerk_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE platforms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL,
    url TEXT NOT NULL,
    name VARCHAR(255), -- Example: 'Trustpilot', 'Amazon'
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (product_id) REFERENCES products(id)
);

CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_id UUID NOT NULL,
    url TEXT UNIQUE NOT NULL,
    author_name VARCHAR(255),
    date_published TIMESTAMP,
    headline TEXT,
    review_body TEXT,
    rating_value DECIMAL(2,1) CHECK (rating_value BETWEEN 0 AND 5),
    language VARCHAR(10), -- Example: 'en', 'fr'
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (platform_id) REFERENCES platforms(id)
);

CREATE TABLE product_stats (
    product_id UUID NOT NULL,
    platform VARCHAR(255) NOT NULL,
    time_period VARCHAR(255) NOT NULL,
    key_highlights TEXT[] NOT NULL,
    pain_points TEXT[] NOT NULL,
    overall_sentiment VARCHAR(1000) NULL,
    sentiment_count TEXT[] NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (product_id, platform, time_period),
    FOREIGN KEY (product_id) REFERENCES products(id)
);

CREATE TABLE google_review_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_id UUID NOT NULL,
    account_id VARCHAR(255) NOT NULL,
    account_name VARCHAR(255) NOT NULL,
    location_id VARCHAR(255) NOT NULL,
    location_name VARCHAR(255) NOT NULL,
    location_address TEXT NOT NULL,
    oauth_token JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (platform_id) REFERENCES platforms(id)
);

ALTER TABLE reviews 
ADD CONSTRAINT reviews_url_unique 
UNIQUE (url);