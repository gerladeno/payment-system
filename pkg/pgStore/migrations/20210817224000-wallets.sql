-- noinspection SqlNoDataSourceInspectionForFile

-- wallets
-- +migrate Up
CREATE TABLE wallet
(
    wallet  uuid                         NOT NULL
        CONSTRAINT wallet_pk PRIMARY KEY,
    amount  numeric(12, 2) DEFAULT 0     NOT NULL CHECK (amount >= 0),
    owner   int                          NOT NULL,
    updated timestamp      DEFAULT NOW() NOT NULL,
    created timestamp      DEFAULT NOW() NOT NULL
);

CREATE TABLE transaction
(
    id              serial                  NOT NULL
        CONSTRAINT transaction_pk PRIMARY KEY,
    type            smallint                NOT NULL,
    wallet          uuid                    NOT NULL,
    wallet_receiver uuid,
    key             char(64) UNIQUE         NOT NULL,
    amount          numeric(12, 2)          NOT NULL,
    ts              timestamp DEFAULT NOW() NOT NULL
);

CREATE TABLE owner
(
    owner   int                     NOT NULL,
    wallet  uuid                    NOT NULL,
    created timestamp DEFAULT NOW() NOT NULL
);

CREATE UNIQUE INDEX owner_wallet_index ON owner (owner, wallet);

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION insert_owner_wallet_f()
    RETURNS TRIGGER
AS
$$
BEGIN
    INSERT INTO owner (owner, wallet, created)
    VALUES (NEW.owner, NEW.wallet, NEW.created)
    ON CONFLICT DO NOTHING;
    RETURN NULL;
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd

DROP TRIGGER IF EXISTS insert_owner_wallet_t
    ON wallet;
CREATE TRIGGER insert_owner_wallet_t
    AFTER INSERT
    ON wallet
    FOR EACH ROW
    EXECUTE PROCEDURE insert_owner_wallet_f();

-- +migrate Down
DROP TABLE wallet CASCADE;
DROP TABLE transaction CASCADE;
DROP TABLE owner CASCADE;