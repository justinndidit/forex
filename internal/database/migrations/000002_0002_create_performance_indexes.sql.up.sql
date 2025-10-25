CREATE INDEX idx_countries_region ON countries (region);
CREATE INDEX idx_countries_currency_code ON countries (currency_code);
CREATE INDEX idx_countries_estimated_gdp_desc ON countries (estimated_gdp DESC);
CREATE INDEX idx_countries_population_desc ON countries (population DESC);