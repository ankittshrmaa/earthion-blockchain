"""Stats API endpoints."""

from fastapi import APIRouter, HTTPException

from app.models import ChainStatsResponse
from app.services.blockchain import get_blockchain_service, BlockchainServiceError

router = APIRouter()


@router.get("/stats", response_model=ChainStatsResponse)
async def get_chain_stats():
    """
    Get blockchain statistics.
    
    Returns:
    - height: Current chain length
    - difficulty: Current mining difficulty
    - totalWork: Cumulative PoW
    - currentReward: Block reward for next block
    - totalMined: Total coins in circulation
    - maxSupply: Maximum possible supply
    """
    try:
        service = get_blockchain_service()
        stats = await service.get_stats()
        return ChainStatsResponse(**stats)
    except BlockchainServiceError as e:
        raise HTTPException(status_code=500, detail=str(e))
