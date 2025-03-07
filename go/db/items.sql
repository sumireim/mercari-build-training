CREATE TABLE IF NOT EXISTS items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    image_name TEXT NOT NULL
);


CREATE TABLE categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,  
    name TEXT NOT NULL                     
);
