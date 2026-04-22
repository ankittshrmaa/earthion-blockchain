"""
Earthion Backend - Blockchain Sync Service
Automatically syncs blockchain data to Supabase.
"""

import asyncio
import logging
from app.services.blockchain import get_blockchain_service
from app.services.supabase import (
    save_transaction, get_transaction_by_txid,
    save_block, get_chain_stats, save_chain_stats
)

logger = logging.getLogger("earthion-sync")

# Track last synced block height
_last_synced_height = 0


async def sync_block_to_supabase(block: dict):
    """Sync a single block to Supabase."""
    try:
        # Save block metadata
        save_block({
            "hash": block.get("hash", ""),
            "height": block.get("index", 0),
            "prev_hash": block.get("prevHash", ""),
            "timestamp": block.get("timestamp", 0),
            "difficulty": block.get("difficulty", 0),
            "tx_count": len(block.get("transactions", []))
        })
        logger.info(f"Synced block {block.get('index')} to Supabase")
        
        # Save transactions
        for tx in block.get("transactions", []):
            txid = tx.get("id", "")
            
            # Check if already synced
            existing = get_transaction_by_txid(txid)
            if existing:
                continue
            
            # Extract from/to from inputs/outputs
            from_addr = ""
            to_addr = ""
            amount = 0
            
            outputs = tx.get("outputs", [])
            if outputs:
                to_addr = outputs[0].get("pubKey", "")
                amount = outputs[0].get("value", 0)
            
            inputs = tx.get("inputs", [])
            if inputs:
                from_addr = inputs[0].get("pubKey", "")
            
            save_transaction({
                "txid": txid,
                "from_address": from_addr,
                "to_address": to_addr,
                "amount": amount,
                "block_height": block.get("index", 0)
            })
            logger.info(f"Synced transaction {txid[:16]}...")
            
    except Exception as e:
        logger.error(f"Error syncing block: {e}")


async def sync_chain():
    """Sync entire chain to Supabase."""
    global _last_synced_height
    
    try:
        bc_service = get_blockchain_service()
        
        # Get current chain height from API
        stats = await bc_service.get_stats()
        current_height = stats.get("height", 0)
        
        if current_height <= _last_synced_height:
            logger.debug(f"No new blocks to sync (height: {current_height})")
            return
        
        # Get blocks from last synced height
        all_blocks = await bc_service.get_all_blocks()
        
        for block in all_blocks:
            block_height = block.get("index", 0)
            if block_height > _last_synced_height:
                await sync_block_to_supabase(block)
        
        _last_synced_height = current_height
        
        # Update chain stats
        save_chain_stats({
            "height": current_height,
            "total_transactions": stats.get("txCount", 0),
            "difficulty": stats.get("difficulty", 0)
        })
        
        logger.info(f"Chain sync complete: {current_height} blocks")
        
    except Exception as e:
        logger.error(f"Error syncing chain: {e}")


async def start_sync_loop(interval: int = 30):
    """
    Start background sync loop.
    Run this in the background to keep Supabase in sync.
    """
    logger.info(f"Starting sync loop (interval: {interval}s)")
    
    # Initial sync
    await sync_chain()
    
    # Periodic sync
    while True:
        await asyncio.sleep(interval)
        await sync_chain()


async def test_connection():
    """Test Supabase connection."""
    try:
        from app.services.supabase import get_supabase_client
        client = get_supabase_client()
        
        # Try a simple query
        result = client.table("blocks").select("count").execute()
        logger.info("✅ Supabase connection successful!")
        return True
    except Exception as e:
        logger.error(f"❌ Supabase connection failed: {e}")
        return False
