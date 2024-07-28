--name: insert-keypair
-- Save hex encoded private key
-- $1: public_key
-- $2: private_key
INSERT INTO keystore(public_key, private_key) VALUES($1, $2) RETURNING id

--name: load-key
-- Load saved key pair
-- $1: public_key
SELECT private_key FROM keystore WHERE public_key=$1