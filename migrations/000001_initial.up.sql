CREATE TABLE IF NOT EXISTS items (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS item_details (
    id INT AUTO_INCREMENT PRIMARY KEY,
    item_id INT NOT NULL,
    note VARCHAR(255) NOT NULL,
    FOREIGN KEY (item_id) REFERENCES items(id)
);

INSERT INTO items (name) VALUES
('Sample Item A'),
('Sample Item B'),
('Sample Item C');

INSERT INTO item_details (item_id, note) VALUES
(1, 'Note for item 1');
