INSERT INTO tenant (uuid, name, document, live, create_at, update_at)
VALUES (gen_random_uuid(), 'Default Tenant', '00000000000', TRUE, NOW(), NOW())
ON CONFLICT (document) DO NOTHING;
