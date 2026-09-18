CREATE TABLE plans (
    plan_id SERIAL PRIMARY KEY,
    duration INTEGER NOT NULL,
    name TEXT NOT NULL UNIQUE
);