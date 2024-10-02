ALTER TABLE otx
ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

CREATE TABLE IF NOT EXISTS dispatch_status_type (
  value TEXT PRIMARY KEY
);
INSERT INTO dispatch_status_type (value) VALUES
('PENDING'),
('IN_NETWORK'),
('SUCCESS'),
('REVERTED'),
('LOW_NONCE'),
('NO_GAS'),
('LOW_GAS_PRICE'),
('NETWORK_ERROR'),
('EXTERNAL_DISPATCH'),
('UNKNOWN_RPC_ERROR');

CREATE TABLE IF NOT EXISTS dispatch (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    otx_id INT REFERENCES otx(id),
    "status" TEXT REFERENCES dispatch_status_type(value) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

create trigger update_dispatch_timestamp
    before update on dispatch
for each row
execute procedure update_timestamp();