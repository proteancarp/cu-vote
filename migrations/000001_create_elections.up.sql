CREATE TABLE elections (
    id text PRIMARY KEY,
    title text NOT NULL,
    opens_at timestamptz NOT NULL,
    closes_at timestamptz NOT NULL,
    state text NOT NULL DEFAULT 'DRAFT',
    CONSTRAINT elections_time_range CHECK (closes_at > opens_at),
    CONSTRAINT elections_state CHECK (state IN ('DRAFT', 'FROZEN', 'OPEN', 'CLOSED', 'FINALIZED'))
);
