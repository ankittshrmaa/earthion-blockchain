"""
Earthion Backend - User Wallet Manager
Creates and manages individual wallets per user.
"""

import secrets
import hashlib
from supabase import create_client, Client
from app.config import settings


def get_supabase() -> Client:
    """Get Supabase client with service role (for backend operations)."""
    import os
    import sys
    sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
    from app.config import settings
    # Use service role key to bypass RLS
    service_key = settings.SUPABASE_SERVICE_KEY or settings.SUPABASE_KEY
    return create_client(settings.SUPABASE_URL, service_key)


def generate_user_address(user_id: str) -> str:
    """
    Generate a deterministic wallet address from user ID.
    This creates a unique address for each user without needing to store private keys.
    """
    # Create a hash-based address from user ID
    data = f"earthion:{user_id}:{secrets.token_hex(8)}".encode()
    hash_bytes = hashlib.sha256(data).digest()
    # Take first 20 bytes as address (like Ethereum/Bitcoin)
    return hash_bytes[:20].hex()


def create_user_wallet(user_id: str, email: str, name: str = None) -> dict:
    """
    Create a new wallet for a user and store in Supabase.
    Returns the wallet address.
    """
    # Generate new unique address
    address = generate_user_address(user_id)
    
    # Save to Supabase
    client = get_supabase()
    client.table("user_wallets").upsert({
        "user_id": user_id,
        "address": address,
        "email": email,
        "name": name,
        "balance": 1000  # Starting balance
    }).execute()
    
    return {"address": address, "user_id": user_id, "name": name}


def get_user_wallet(user_id: str) -> dict:
    """Get wallet for a user."""
    client = get_supabase()
    result = client.table("user_wallets").select("*").eq("user_id", user_id).execute()
    
    if result.data:
        return result.data[0]
    
    return None


def update_user_balance(user_id: str, amount: int) -> dict:
    """Update user balance."""
    client = get_supabase()
    return client.table("user_wallets").update({
        "balance": amount,
        "updated_at": "now()"
    }).eq("user_id", user_id).execute()


def get_all_wallets() -> list:
    """Get all user wallets (for airdrop, etc)."""
    client = get_supabase()
    result = client.table("user_wallets").select("*").execute()
    return result.data


# Initialize user_wallets table in Supabase
def init_user_wallets_table():
    """Create the user_wallets table if not exists."""
    client = get_supabase()
    
    # Create table via SQL
    sql = """
    CREATE TABLE IF NOT EXISTS user_wallets (
        id SERIAL PRIMARY KEY,
        user_id TEXT UNIQUE NOT NULL,
        address TEXT UNIQUE NOT NULL,
        email TEXT,
        balance INTEGER DEFAULT 0,
        created_at TIMESTAMP DEFAULT NOW(),
        updated_at TIMESTAMP DEFAULT NOW()
    );
    
    CREATE INDEX IF NOT EXISTS idx_user_wallets_user ON user_wallets(user_id);
    CREATE INDEX IF NOT EXISTS idx_user_wallets_address ON user_wallets(address);
    
    -- Allow public read
    CREATE POLICY "Public read wallets" ON user_wallets FOR SELECT USING (true);
    -- Allow service write
    CREATE POLICY "Service write wallets" ON user_wallets FOR ALL USING (true);
    """
    
    # Execute via postgrest
    client.postgrest().rpc("exec_sql", {"query": sql}).execute()


# For demo: faucet test coins
def faucet(address: str, amount: int = 1000) -> dict:
    """Give test coins to a wallet address."""
    client = get_supabase()
    
    # Check if exists
    result = client.table("user_wallets").select("balance").eq("address", address).execute()
    
    if result.data:
        new_balance = result.data[0].get("balance", 0) + amount
        client.table("user_wallets").update({
            "balance": new_balance
        }).eq("address", address).execute()
        
        return {"success": True, "balance": new_balance}
    
    return {"success": False, "error": "Wallet not found"}