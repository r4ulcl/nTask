CREATE TABLE IF NOT EXISTS worker (
    name VARCHAR(255) PRIMARY KEY,
    ip VARCHAR(255) NOT NULL,
    port VARCHAR(255) NOT NULL,
    oauthToken VARCHAR(255) NOT NULL,
    IddleThreads INT,
    up BOOLEAN,
    downCount INT,
    UNIQUE (ip, port)
);

CREATE TABLE IF NOT EXISTS task (
    ID VARCHAR(255) PRIMARY KEY,
    command TEXT,
    name TEXT,
    createdAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updatedAt TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    executedAt TIMESTAMP NOT NULL DEFAULT '1970-01-01 00:00:01',
    status VARCHAR(255), 
    workerName VARCHAR(255),
    username VARCHAR(255),
    priority INT DEFAULT 0,
    callbackURL TEXT,
    callbackToken TEXT
);
