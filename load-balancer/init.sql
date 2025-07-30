-- init.sql
CREATE TABLE messages (
    id SERIAL PRIMARY KEY,
    message VARCHAR(255) NOT NULL
);

INSERT INTO messages (message) VALUES
('A'),
('B'),
('C'),
('D');