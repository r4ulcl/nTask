CREATE TABLE IF NOT EXISTS worker (
    name VARCHAR(255) PRIMARY KEY,
    ip VARCHAR(255) NOT NULL,
    port VARCHAR(255) NOT NULL,
    working BOOLEAN,
    up BOOLEAN,
    UNIQUE (ip, port)
);

CREATE TABLE IF NOT EXISTS task (
    ID VARCHAR(255) PRIMARY KEY,
    date VARCHAR(255),
    status VARCHAR(255),
    workerName VARCHAR(255)
);
