CREATE TABLE IF NOT EXISTS countries (
    id SERIAL PRIMARY KEY,
    name VARCHAR(256) NOT NULL,
    capital VARCHAR(256),
    region VARCHAR(256),
    population BIGINT NOT NULL,
    currency_code VARCHAR(20),
    exchange_rate DECIMAL(15, 6),
    estimated_gdp DECIMAL(20, 2),
    flag_url VARCHAR(256),
    last_refreshed_at TIMESTAMP NOT NULL,

    UNIQUE (name)
);

CREATE TABLE IF NOT EXISTS app_status (
    id INT PRIMARY KEY DEFAULT 1,
    last_refreshed_at TIMESTAMP
);

INSERT INTO app_status (id, last_refreshed_at)
VALUES (1, NULL)
ON CONFLICT (id) DO NOTHING;