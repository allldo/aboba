CREATE TYPE manga_status AS ENUM ('ongoing', 'completed', 'hiatus', 'cancelled');

CREATE TABLE manga (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    author VARCHAR(255) NOT NULL,
    artist VARCHAR(255),
    genres JSONB DEFAULT '[]',
    status manga_status NOT NULL DEFAULT 'ongoing',
    year INTEGER,
    chapters INTEGER DEFAULT 0,
    price DECIMAL(10,2) NOT NULL DEFAULT 0,
    cover_image VARCHAR(500),
    stock INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_manga_title ON manga(title);
CREATE INDEX idx_manga_author ON manga(author);
CREATE INDEX idx_manga_genres ON manga USING GIN(genres);
CREATE INDEX idx_manga_status ON manga(status);
CREATE INDEX idx_manga_active ON manga(is_active);