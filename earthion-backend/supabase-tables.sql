-- ============================================
-- EARTHION SUPERBASE DATABASE SCHEMA
-- ============================================

-- 1. USER WALLETS (one wallet per user)
-- ============================================
DROP TABLE IF EXISTS user_wallets CASCADE;

CREATE TABLE user_wallets (
    id SERIAL PRIMARY KEY,
    user_id TEXT UNIQUE NOT NULL,
    address TEXT UNIQUE NOT NULL,
    email TEXT,
    balance INTEGER DEFAULT 1000,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_user_wallets_user ON user_wallets(user_id);
CREATE INDEX idx_user_wallets_address ON user_wallets(address);

ALTER TABLE user_wallets ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Users can read own wallet" ON user_wallets 
    FOR SELECT USING (auth.uid()::text = user_id);
CREATE POLICY "Users can insert own wallet" ON user_wallets 
    FOR INSERT WITH CHECK (auth.uid()::text = user_id);
CREATE POLICY "Users can update own wallet" ON user_wallets 
    FOR UPDATE USING (auth.uid()::text = user_id);


-- 2. TRANSACTIONS (blockchain transactions)
-- ============================================
DROP TABLE IF EXISTS transactions CASCADE;

CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    txid TEXT UNIQUE NOT NULL,
    from_address TEXT,
    to_address TEXT,
    amount INTEGER NOT NULL,
    fee INTEGER DEFAULT 0,
    block_height INTEGER,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_transactions_txid ON transactions(txid);
CREATE INDEX idx_transactions_from ON transactions(from_address);
CREATE INDEX idx_transactions_to ON transactions(to_address);
CREATE INDEX idx_transactions_block ON transactions(block_height);

ALTER TABLE transactions ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Public read transactions" ON transactions FOR SELECT USING (true);
CREATE POLICY "Public insert transactions" ON transactions FOR INSERT WITH CHECK (true);


-- 3. BLOCKS
-- ============================================
DROP TABLE IF EXISTS blocks CASCADE;

CREATE TABLE blocks (
    id SERIAL PRIMARY KEY,
    hash TEXT UNIQUE NOT NULL,
    height INTEGER NOT NULL,
    prev_hash TEXT,
    timestamp BIGINT,
    difficulty INTEGER,
    tx_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_blocks_height ON blocks(height);
CREATE INDEX idx_blocks_hash ON blocks(hash);

ALTER TABLE blocks ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Public read blocks" ON blocks FOR SELECT USING (true);
CREATE POLICY "Public insert blocks" ON blocks FOR INSERT WITH CHECK (true);


-- 4. CHAIN STATS
-- ============================================
DROP TABLE IF EXISTS chain_stats CASCADE;

CREATE TABLE chain_stats (
    id SERIAL PRIMARY KEY,
    height INTEGER NOT NULL,
    total_transactions INTEGER DEFAULT 0,
    difficulty INTEGER,
    created_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE chain_stats ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Public read chain_stats" ON chain_stats FOR SELECT USING (true);
CREATE POLICY "Public insert chain_stats" ON chain_stats FOR INSERT WITH CHECK (true);


-- 5. NFTS
-- ============================================
DROP TABLE IF EXISTS nfts CASCADE;

CREATE TABLE nfts (
    id SERIAL PRIMARY KEY,
    owner_id TEXT NOT NULL,
    token_id TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    image_url TEXT,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

ALTER TABLE nfts ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Public read nfts" ON nfts FOR SELECT USING (true);
CREATE POLICY "Users can insert nfts" ON nfts FOR INSERT WITH CHECK (true);
CREATE POLICY "Users can update own nfts" ON nfts FOR UPDATE USING (true);

-- ============================================
-- TABLES CREATED SUCCESSFULLY!
-- ============================================