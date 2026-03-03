CREATE TABLE IF NOT EXISTS alerts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token           TEXT         NOT NULL,
    target_price    NUMERIC(20,8) NOT NULL,
    direction       TEXT         NOT NULL CHECK (direction IN ('above', 'below')),
    callback_url    TEXT         NOT NULL,
    callback_secret TEXT         NOT NULL,
    triggered       BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_alerts_active ON alerts (triggered) WHERE triggered = FALSE;
