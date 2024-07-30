-- Keystore
CREATE TABLE IF NOT EXISTS keystore (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_key TEXT NOT NULL,
    private_key TEXT NOT NULL,
    active BOOLEAN DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS public_key_idx ON keystore(public_key);

-- Master key
CREATE TABLE IF NOT EXISTS master_key (
    id INT PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Noncestore
CREATE TABLE IF NOT EXISTS noncestore (
    keystore_id INT PRIMARY KEY REFERENCES keystore(id),
    next_nonce INT DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- OTX tx type enum table
CREATE TABLE IF NOT EXISTS otx_tx_type (
  value TEXT PRIMARY KEY
);
INSERT INTO otx_tx_type (value) VALUES
('GAS_REFILL'),
('ACCOUNT_REGISTER'),
('GAS_TRANSFER'),
('TOKEN_TRANSFER'),
('TOKEN_APPROVE'),
('POOL_SWAP'),
('POOL_DEPOSIT'),
('OTHER_MANUAL');

-- OTX
CREATE TABLE IF NOT EXISTS otx (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tracking_id uuid NOT NULL,
    otx_type TEXT REFERENCES otx_tx_type(value) NOT NULL,
    signer_account INT REFERENCES keystore(id) NOT NULL,
    raw_tx TEXT NOT NULL,
    tx_hash TEXT NOT NULL,
    nonce INT NOT NULL,
    replaced BOOLEAN DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- nonce trigger on keystore insert to default 0
create function insert_starting_nonce()
    returns trigger
as $$
begin
    insert into noncestore (keystore_id) values (new.id);
    return new;
end;
$$ language plpgsql;

create trigger insert_starting_nonce
    after insert on keystore
for each row
execute procedure insert_starting_nonce();

-- updated_at function
create function update_timestamp()
    returns trigger
as $$
begin
    new.updated_at = current_timestamp;
    return new;
end;
$$ language plpgsql;

create trigger update_keystore_timestamp
    before update on keystore
for each row
execute procedure update_timestamp();

create trigger update_master_key_timestamp
    before update on master_key
for each row
execute procedure update_timestamp();

create trigger update_nonce_timestamp
    before update on noncestore
for each row
execute procedure update_timestamp();
