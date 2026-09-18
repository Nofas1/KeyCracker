CREATE TABLE subscriptions (
    subs_id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    plan_id INTEGER NOT NULL REFERENCES plans(plan_id),
    start_at TIMESTAMPTZ NOT NULL,
    expired_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL CHECK(
        status IN ('active', 'expired', 'canceled')
    )
);