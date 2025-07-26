-- DDL for items table
CREATE TABLE IF NOT EXISTS items (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

-- Example data
INSERT INTO items (name) VALUES
('Sample Item A'),
('Sample Item B'),
('Sample Item C');
