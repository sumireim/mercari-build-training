CREATE TABLE IF NOT EXISTS items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    category_id INTEGER NOT NULL,
    image_name TEXT NOT NULL
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
);


CREATE TABLE IF NOT EXISTS categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,  
    name TEXT NOT NULL                     
);

INSERT INTO categories (name) VALUES ('phone');

INSERT INTO categories (name) VALUES ('fashion');