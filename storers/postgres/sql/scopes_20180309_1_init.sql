-- +migrate Up
CREATE TABLE scopes (
	id VARCHAR(36) PRIMARY KEY,
	user_policy VARCHAR NOT NULL DEFAULT '',
	user_exceptions VARCHAR[] NOT NULL,
	client_policy vARCHAR NOT NULL DEFAULT '',
	client_exceptions VARCHAR[] NOT NULL,
	is_default BOOLEAN NOT NULL DEFAULT false
);

-- +migrate Down
DROP TABLE scopes;
