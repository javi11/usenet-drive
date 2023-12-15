-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS corrupted_segments (
			id INTEGER PRIMARY KEY,
			uuid TEXT UNIQUE,
			number INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            FOREIGN KEY (nzbId) REFERENCES corrupted_nzbs(nzbId)
		);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE corrupted_segments;
-- +goose StatementEnd
