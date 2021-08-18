-- noinspection SqlNoDataSourceInspectionForFile

-- wallets
-- +migrate Up
CREATE TABLE wallet
(
    wallet  uuid                         NOT NULL
        CONSTRAINT wallet_pk PRIMARY KEY,
    amount  numeric(12, 2) DEFAULT 0     NOT NULL CHECK (amount >= 0),
    owner   int                          NOT NULL,
    status  smallint       DEFAULT 0     NOT NULL,
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
    key             text UNIQUE             NOT NULL,
    amount          numeric(12, 2)          NOT NULL,
    ts              timestamp DEFAULT NOW() NOT NULL
);

CREATE INDEX transaction_wallet_index ON transaction (wallet);
CREATE INDEX transaction_wallet_receiver_index ON transaction (wallet_receiver) WHERE wallet_receiver IS NOT NULL;

-- +migrate Down
DROP TABLE wallet CASCADE;
DROP TABLE transaction CASCADE;