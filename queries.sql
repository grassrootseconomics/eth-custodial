--name: insert-keypair
-- Save hex encoded private key
-- $1: public_key
-- $2: private_key
INSERT INTO keystore(public_key, private_key) VALUES($1, $2) RETURNING id;

--name: load-key
-- Load saved key pair
-- $1: public_key
SELECT private_key FROM keystore WHERE public_key=$1;

--name: load-master-key
-- Load saved master key pair
SELECT private_key FROM keystore
INNER JOIN (SELECT id FROM master_key LIMIT 1) master_key ON keystore.id = master_key.id;

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

--name: peek-nonce
-- Get current nonce
-- $1: public_key
WITH peek_nonce AS (
    SELECT 
        CASE 
            WHEN next_nonce = 0 THEN 0 
            ELSE next_nonce - 1 
        END AS current_nonce
    FROM noncestore
    WHERE keystore_id = (
        SELECT id FROM keystore WHERE public_key = $1
    )
)
SELECT COALESCE(current_nonce, 0) AS current_nonce
FROM peek_nonce;

--name: acquire-nonce
-- Fetch next nonce and decrement by 1 to get the current nonce
-- $1: public_key
WITH updated_nonce AS (
    UPDATE noncestore
    SET next_nonce = next_nonce + 1
    WHERE keystore_id = (
        SELECT id FROM keystore WHERE public_key = $1
    )
    RETURNING next_nonce - 1 AS current_nonce
)
SELECT current_nonce FROM updated_nonce;

--name: set-account-nonce
-- Set accout nonce manually
-- $1: public_key
-- $2: nonce_value
UPDATE noncestore
SET next_nonce = $2
WHERE keystore_id = (
    SELECT id FROM keystore WHERE public_key = $1
);
