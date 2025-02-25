-- Schema for `sites` table
-- data_cid ipfs hash (pointer to data) will be moved to blockchain
CREATE TABLE public.sites (
    domain   TEXT NOT NULL,
    owner    TEXT,
    data     JSONB,
    status   TEXT,
    preview  JSONB,
    data_cid TEXT,
    CONSTRAINT sites_pkey PRIMARY KEY (domain)
);