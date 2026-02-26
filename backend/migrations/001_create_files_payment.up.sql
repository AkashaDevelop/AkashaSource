-- +dialect:sqlite
CREATE TABLE IF NOT EXISTS stored_files (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER,
  purpose TEXT,
  filename TEXT,
  bytes INTEGER,
  content BLOB,
  created_at INTEGER
);
CREATE TABLE IF NOT EXISTS payment_orders (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id INTEGER,
  amount REAL,
  status INTEGER,
  provider TEXT,
  pay_url TEXT,
  notify_data TEXT,
  created_at INTEGER,
  paid_at INTEGER
);

-- +dialect:mysql
CREATE TABLE IF NOT EXISTS stored_files (
  id INT AUTO_INCREMENT PRIMARY KEY,
  user_id INT,
  purpose VARCHAR(255),
  filename VARCHAR(255),
  bytes BIGINT,
  content LONGBLOB,
  created_at BIGINT
);
CREATE TABLE IF NOT EXISTS payment_orders (
  id INT AUTO_INCREMENT PRIMARY KEY,
  user_id INT,
  amount DOUBLE,
  status INT,
  provider VARCHAR(255),
  pay_url TEXT,
  notify_data TEXT,
  created_at BIGINT,
  paid_at BIGINT
);

-- +dialect:postgres
CREATE TABLE IF NOT EXISTS stored_files (
  id SERIAL PRIMARY KEY,
  user_id INTEGER,
  purpose VARCHAR(255),
  filename VARCHAR(255),
  bytes BIGINT,
  content BYTEA,
  created_at BIGINT
);
CREATE TABLE IF NOT EXISTS payment_orders (
  id SERIAL PRIMARY KEY,
  user_id INTEGER,
  amount DOUBLE PRECISION,
  status INTEGER,
  provider VARCHAR(255),
  pay_url TEXT,
  notify_data TEXT,
  created_at BIGINT,
  paid_at BIGINT
);
