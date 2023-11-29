CREATE TABLE IF NOT EXISTS worker (
    name VARCHAR(255) PRIMARY KEY,
    ip VARCHAR(255) NOT NULL,
    port VARCHAR(255) NOT NULL,
    data JSON,
    UNIQUE (ip, port)
);
