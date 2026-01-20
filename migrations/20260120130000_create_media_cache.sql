-- +goose Up
-- +goose StatementBegin
-- Create media_cache table for storing media metadata for recommendations
CREATE TABLE media_cache (
    id INTEGER PRIMARY KEY, -- AniList media ID
    type TEXT NOT NULL, -- 'ANIME' or 'MANGA'
    title_romaji TEXT NOT NULL,
    title_english TEXT,
    cover_image_large TEXT,
    cover_image_medium TEXT,

    -- Recommendation-relevant fields
    genres TEXT[], -- Array of genre strings for similarity matching
    average_score INTEGER, -- Quality indicator
    mean_score INTEGER, -- Alternative quality metric
    popularity INTEGER, -- Popularity for weighting recommendations
    favourites INTEGER DEFAULT 0, -- User favorite count
    format TEXT, -- TV, MOVIE, MANGA, ONE_SHOT, etc. - user format preferences
    status TEXT, -- FINISHED, RELEASING, NOT_YET_RELEASED, CANCELLED, HIATUS
    season_year INTEGER, -- Recency/era preferences
    season TEXT, -- WINTER, SPRING, SUMMER, FALL
    is_adult BOOLEAN DEFAULT FALSE, -- Content filtering

    -- Metadata
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create tags table for unique tag names
CREATE TABLE tags (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

-- Create media_tags junction table
CREATE TABLE media_tags (
    media_id INTEGER NOT NULL REFERENCES media_cache(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    rank INTEGER NOT NULL, -- Tag relevance/importance for weighting
    is_media_spoiler BOOLEAN DEFAULT FALSE,
    is_general_spoiler BOOLEAN DEFAULT FALSE,
    PRIMARY KEY (media_id, tag_id)
);

-- Create indexes for common queries
CREATE INDEX idx_media_cache_type ON media_cache(type);
CREATE INDEX idx_media_cache_genres ON media_cache USING GIN(genres);
CREATE INDEX idx_media_cache_average_score ON media_cache(average_score);
CREATE INDEX idx_media_cache_popularity ON media_cache(popularity);
CREATE INDEX idx_media_cache_season_year ON media_cache(season_year);
CREATE INDEX idx_media_cache_format ON media_cache(format);

-- Indexes for tag queries (for finding similar media by tags)
CREATE INDEX idx_media_tags_tag_id ON media_tags(tag_id);
CREATE INDEX idx_media_tags_rank ON media_tags(rank);
CREATE INDEX idx_tags_name ON tags(name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE media_tags;
DROP TABLE tags;
DROP TABLE media_cache;
-- +goose StatementEnd
