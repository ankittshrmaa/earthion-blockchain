"""Blocks API endpoints."""

from fastapi import APIRouter, HTTPException, Query
from typing import Optional, Any

from app.services.blockchain import get_blockchain_service, BlockchainServiceError

router = APIRouter()


@router.get("")
async def get_blocks(limit: Optional[int] = Query(None, ge=1, le=1000)):
    """
    Get all blocks.
    
    Optional pagination with limit parameter.
    """
    try:
        service = get_blockchain_service()
        blocks = await service.get_all_blocks()
        
        if limit:
            blocks = blocks[:limit]
        
        # Transform blocks to include txCount
        result_blocks = []
        for b in blocks:
            txns = b.get('transactions', [])
            result_blocks.append({
                'index': b.get('index'),
                'timestamp': b.get('timestamp'),
                'prevHash': b.get('prevHash'),
                'merkleRoot': b.get('merkleRoot'),
                'hash': b.get('hash'),
                'nonce': b.get('nonce'),
                'difficulty': b.get('difficulty'),
                'txCount': len(txns) if txns else 0,
                'transactions': txns
            })
        
        return {
            'blocks': result_blocks,
            'count': len(result_blocks)
        }
    except BlockchainServiceError as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/{block_hash}")
async def get_block(block_hash: str):
    """
    Get block by hash.
    """
    try:
        service = get_blockchain_service()
        block = await service.get_block_by_hash(block_hash)
        if not block:
            raise HTTPException(status_code=404, detail="Block not found")
        return block
    except BlockchainServiceError as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/index/{index}")
async def get_block_by_index(index: int):
    """
    Get block by index.
    """
    if index < 0:
        raise HTTPException(status_code=400, detail="Index must be non-negative")
    
    try:
        service = get_blockchain_service()
        block = await service.get_block_by_index(index)
        if not block:
            raise HTTPException(status_code=404, detail="Block not found")
        return block
    except BlockchainServiceError as e:
        raise HTTPException(status_code=500, detail=str(e))
