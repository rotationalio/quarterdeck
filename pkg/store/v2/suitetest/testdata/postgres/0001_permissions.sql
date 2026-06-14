-- Insert test data of roles and permissions
INSERT INTO roles (id, title, description, is_default, created, modified) VALUES
    (1, 'admin', 'Administrator role with all permissions', false, '2025-02-14T11:21:42+00:00', '2025-02-14T11:21:42+00:00'),
    (2, 'editor', 'Editor role with permissions to create and edit content', true, '2025-02-14T11:21:42+00:00', '2025-02-14T11:21:42+00:00'),
    (3, 'viewer', 'Viewer role with permissions to view content only', false, '2025-02-14T11:21:42+00:00', '2025-02-14T11:21:42+00:00'),
    (4, 'keyholder', 'Keyholder role with permissions to manage API keys', false, '2025-02-14T11:21:42+00:00', '2025-02-14T11:21:42+00:00')
;

INSERT INTO permissions (id, title, description, created, modified) VALUES
    (1, 'content:modify', 'Permission to create and edit content', '2025-02-14T11:21:42+00:00', '2025-02-14T11:21:42+00:00'),
    (2, 'content:view', 'Permission to view content', '2025-02-14T11:21:42+00:00', '2025-02-14T11:21:42+00:00'),
    (3, 'content:delete', 'Permission to delete content', '2025-02-14T11:21:42+00:00', '2025-02-14T11:21:42+00:00'),
    (4, 'users:view', 'Permission to view users', '2025-02-14T11:21:42+00:00', '2025-02-14T11:21:42+00:00'),
    (5, 'users:invite', 'Permission to invite new users', '2025-02-14T11:21:42+00:00', '2025-02-14T11:21:42+00:00'),
    (6, 'users:delete', 'Permission to delete user accounts', '2025-02-14T11:21:42+00:00', '2025-02-14T11:21:42+00:00'),
    (7, 'users:modify', 'Permission to change other user accounts', '2025-02-14T11:21:42+00:00', '2025-02-14T11:21:42+00:00'),
    (8, 'keys:create', 'Permission to create api keys', '2025-02-14T11:21:42+00:00', '2025-02-14T11:21:42+00:00'),
    (9, 'keys:revoke', 'Permission to revoke api keys', '2025-02-14T11:21:42+00:00', '2025-02-14T11:21:42+00:00'),
    (10, 'keys:view', 'Permission to view api keys', '2025-02-14T11:21:42+00:00', '2025-02-14T11:21:42+00:00')
;

INSERT INTO role_permissions (role_id, permission_id, created) VALUES
    (1, 1, '2025-02-14T11:21:42+00:00'),
    (1, 2, '2025-02-14T11:21:42+00:00'),
    (1, 3, '2025-02-14T11:21:42+00:00'),
    (1, 4, '2025-02-14T11:21:42+00:00'),
    (1, 5, '2025-02-14T11:21:42+00:00'),
    (1, 6, '2025-02-14T11:21:42+00:00'),
    (1, 7, '2025-02-14T11:21:42+00:00'),
    (1, 8, '2025-02-14T11:21:42+00:00'),
    (1, 9, '2025-02-14T11:21:42+00:00'),
    (1, 10, '2025-02-14T11:21:42+00:00'),
    (2, 1, '2025-02-14T11:21:42+00:00'),
    (2, 2, '2025-02-14T11:21:42+00:00'),
    (2, 3, '2025-02-14T11:21:42+00:00'),
    (2, 4, '2025-02-14T11:21:42+00:00'),
    (2, 5, '2025-02-14T11:21:42+00:00'),
    (2, 10, '2025-02-14T11:21:42+00:00'),
    (3, 2, '2025-02-14T11:21:42+00:00'),
    (3, 4, '2025-02-14T11:21:42+00:00'),
    (4, 4, '2025-02-14T11:21:42+00:00'),
    (4, 8, '2025-02-14T11:21:42+00:00'),
    (4, 9, '2025-02-14T11:21:42+00:00'),
    (4, 10, '2025-02-14T11:21:42+00:00')
;

SELECT setval('roles_id_seq', (SELECT MAX(id) FROM roles));
SELECT setval('permissions_id_seq', (SELECT MAX(id) FROM permissions));
