-- Write your migrate up statements here
-- Index for the ?region filter
CREATE INDEX IF NOT EXISTS idx_countries_region ON countries (region);

-- Index for the ?currency filter
CREATE INDEX IF NOT EXISTS idx_countries_currency_code ON countries (currency_code);

-- Index for the ?sort=gdp_desc filter (the most likely sort)
CREATE INDEX IF NOT EXISTS idx_countries_estimated_gdp_desc ON countries (estimated_gdp DESC NULLS LAST);

-- Index for population sorting
CREATE INDEX IF NOT EXISTS idx_countries_population_desc ON countries (population DESC);

---- create above / drop below ----

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
