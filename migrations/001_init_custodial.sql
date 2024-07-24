CREATE TABLE IF NOT EXISTS keystore (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_key TEXT NOT NULL,
    private_key TEXT NOT NULL,
    active BOOLEAN DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS public_key_idx ON keystore(public_key);

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