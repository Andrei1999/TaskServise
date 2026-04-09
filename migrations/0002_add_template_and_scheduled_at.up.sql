CREATE TABLE IF NOT EXISTS task_templates (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    rule_type TEXT NOT NULL,
    rule_params JSONB NOT NULL,
    execution_time TIME NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE tasks ADD COLUMN scheduled_at TIMESTAMPTZ;
ALTER TABLE tasks ADD COLUMN template_id BIGINT REFERENCES task_templates(id) ON DELETE CASCADE;

CREATE INDEX idx_tasks_template_id ON tasks(template_id);
