-- Deterministic UUIDs for repeatable seed data (dev / tests).
-- Password: password123 (bcrypt cost 12).

INSERT INTO users (id, name, email, password, created_at)
VALUES (
    '10000000-0000-4000-8000-000000000001',
    'Test User',
    'test@example.com',
    '$2a$12$GIBGEpJv4SQWgdWpzGMTbePjkwRxFtNkQT2pID4Wk/X2J97LEMH6a',
    CURRENT_TIMESTAMP
);

INSERT INTO projects (id, name, description, owner_id, created_at)
VALUES (
    '20000000-0000-4000-8000-000000000001',
    'Demo Project',
    'Seed project owned by test@example.com',
    '10000000-0000-4000-8000-000000000001',
    CURRENT_TIMESTAMP
);

INSERT INTO tasks (
    id,
    title,
    description,
    status,
    priority,
    project_id,
    assignee_id,
    due_date,
    created_at,
    updated_at
)
VALUES
    (
        '30000000-0000-4000-8000-000000000001',
        'Backlog item',
        'Task in todo state',
        'todo',
        'low',
        '20000000-0000-4000-8000-000000000001',
        '10000000-0000-4000-8000-000000000001',
        NULL,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        '30000000-0000-4000-8000-000000000002',
        'Active work',
        'Task currently in progress',
        'in_progress',
        'medium',
        '20000000-0000-4000-8000-000000000001',
        '10000000-0000-4000-8000-000000000001',
        NULL,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    ),
    (
        '30000000-0000-4000-8000-000000000003',
        'Shipped feature',
        'Completed task',
        'done',
        'high',
        '20000000-0000-4000-8000-000000000001',
        NULL,
        NULL,
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP
    );
