INSERT INTO candidates (id, name) VALUES
    ('candA', 'Candidate A'),
    ('candB', 'Candidate B'),
    ('candC', 'Candidate C')
ON CONFLICT (id) DO NOTHING;
