CREATE TABLE IF NOT EXISTS countries (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(256) NOT NULL,
    capital VARCHAR(256),
    region VARCHAR(256),
    population BIGINT NOT NULL,
    currency_code VARCHAR(20),
    exchange_rate DECIMAL(15, 6),
    estimated_gdp DECIMAL(20, 2),
    flag_url VARCHAR(256),
    last_refreshed_at TIMESTAMP NOT NULL,
    UNIQUE KEY (name)
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci; -- <-- This is the fix

CREATE TABLE IF NOT EXISTS app_status (
    id INT PRIMARY KEY DEFAULT 1,
    last_refreshed_at TIMESTAMP NULL
) DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci; -- <-- This is the fix

INSERT IGNORE INTO app_status (id, last_refreshed_at)
VALUES (1, NULL);