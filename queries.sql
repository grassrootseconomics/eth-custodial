--name: insert-keypair
-- Save hex encoded private key
-- $1: public_key
-- $2: private_key
INSERT INTO keystore(public_key, private_key) VALUES($1, $2) RETURNING id

--name: load-key
-- Load saved key pair
-- $1: public_key
SELECT private_key FROM keystore WHERE public_key=$1

--name: load-master-key
-- Load saved master key pair
SELECT private_key FROM keystore
INNER JOIN (SELECT id FROM master_key LIMIT 1) master_key ON keystore.id = master_key.id

--name: bootstrap-master-key
-- Save newely hex encoded private key to be used as a master key
-- $1: public_key
-- $2: private_key
WITH new_key AS (
    INSERT INTO keystore (public_key, private_key, active)
    SELECT $1, $2, true
    WHERE NOT EXISTS (
        SELECT 1 FROM master_key
    )
    RETURNING id
)
INSERT INTO master_key (id)
SELECT id FROM new_key
RETURNING id;

