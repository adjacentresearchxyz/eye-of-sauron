-- Create a dedicated duplicates table for enhanced deduplication
CREATE TABLE IF NOT EXISTS duplicates (
    id SERIAL PRIMARY KEY,
    original_title TEXT NOT NULL,
    cleaned_title TEXT NOT NULL UNIQUE,
    link TEXT NOT NULL,
    first_seen_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes to speed up similarity searches
CREATE INDEX IF NOT EXISTS idx_duplicates_cleaned_title ON duplicates (cleaned_title);
CREATE INDEX IF NOT EXISTS idx_duplicates_first_seen_at ON duplicates (first_seen_at);

-- Add a new column to sources table to track filtered status
ALTER TABLE sources ADD COLUMN IF NOT EXISTS filtered_out BOOLEAN DEFAULT FALSE;
ALTER TABLE sources ADD COLUMN IF NOT EXISTS filter_reason TEXT;

-- Add column for tracking duplicate relationship
ALTER TABLE sources ADD COLUMN IF NOT EXISTS duplicate_of INTEGER REFERENCES sources(id);