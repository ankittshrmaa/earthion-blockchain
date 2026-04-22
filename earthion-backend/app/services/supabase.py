"""
Earthion Backend - Supabase Service
Handles database operations via Supabase (PostgreSQL).
"""

from typing import Optional
from supabase import create_client, Client
from app.config import settings


# Singleton instance
_supabase_client: Optional[Client] = None


def get_supabase_client() -> Client:
    """Get or create Supabase client with service role."""
    global _supabase_client
    if _supabase_client is None:
        # Use service role to bypass RLS
        service_key = settings.SUPABASE_SERVICE_KEY or settings.SUPABASE_KEY
        _supabase_client = create_client(
            settings.SUPABASE_URL,
            service_key
        )
    return _supabase_client


# ========== Database Operations ==========

def save_transaction(tx_data: dict) -> dict:
    """Save transaction to Supabase."""
    client = get_supabase_client()
    return client.table("transactions").insert(tx_data).execute()


def get_transactions(limit: int = 100) -> list:
    """Get all transactions."""
    client = get_supabase_client()
    return client.table("transactions").select("*").limit(limit).execute().data


def get_transaction_by_txid(txid: str) -> Optional[dict]:
    """Get transaction by TXID."""
    client = get_supabase_client()
    result = client.table("transactions").select("*").eq("txid", txid).execute()
    return result.data[0] if result.data else None


def save_block(block_data: dict) -> dict:
    """Save block metadata to Supabase."""
    client = get_supabase_client()
    return client.table("blocks").insert(block_data).execute()


def get_blocks(limit: int = 50) -> list:
    """Get blocks."""
    client = get_supabase_client()
    return client.table("blocks").select("*").order("height", desc=True).limit(limit).execute().data


def save_wallet_balance(address: str, balance: int) -> dict:
    """Save wallet balance."""
    client = get_supabase_client()
    return client.table("wallets").upsert({
        "address": address,
        "balance": balance
    }).execute()


def get_wallet_balance(address: str) -> Optional[dict]:
    """Get wallet balance."""
    client = get_supabase_client()
    result = client.table("wallets").select("*").eq("address", address).execute()
    return result.data[0] if result.data else None


def save_chain_stats(stats: dict) -> dict:
    """Save chain statistics."""
    client = get_supabase_client()
    return client.table("chain_stats").insert(stats).execute()


def get_chain_stats() -> Optional[dict]:
    """Get latest chain statistics."""
    client = get_supabase_client()
    result = client.table("chain_stats").select("*").order("created_at", desc=True).limit(1).execute()
    return result.data[0] if result.data else None
