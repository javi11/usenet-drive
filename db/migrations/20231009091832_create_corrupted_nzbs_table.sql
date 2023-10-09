-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS corrupted_nzbs (
			id INTEGER PRIMARY KEY,
			path TEXT UNIQUE,
			error TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE corrupted_nzbs;
-- +goose StatementEnd
