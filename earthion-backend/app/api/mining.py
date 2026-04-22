"""Mining API endpoints."""

from fastapi import APIRouter, HTTPException, Request
from slowapi import Limiter
from slowapi.util import get_remote_address
from slowapi.errors import RateLimitExceeded

from app.models import MineBlockResponse
from app.services.blockchain import get_blockchain_service, BlockchainServiceError

router = APIRouter()

# Rate limiter for mining
limiter = Limiter(key_func=get_remote_address)


@router.post("/mine", response_model=MineBlockResponse)
@limiter.limit("5/minute")  # Limit mining to 5 per minute
async def mine_block(request: Request):
    """
    Mine a new block.
    
    Creates a new block with a coinbase transaction to the wallet address.
    Rate limited to 1 request per minute.
    """
    try:
        service = get_blockchain_service()
        block_data = await service.mine_block()
        
        # Get current reward from stats
        stats = await service.get_stats()
        reward = stats.get("currentReward", 0)
        
        return MineBlockResponse(
            block=block_data,
            reward=reward
        )
    except RateLimitExceeded:
        raise HTTPException(status_code=429, detail="Rate limit exceeded. Try again later.")
    except BlockchainServiceError as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/reward")
async def get_current_reward():
    """
    Get current block reward.
    """
    try:
        service = get_blockchain_service()
        stats = await service.get_stats()
        return {
            "reward": stats.get("currentReward", 0),
            "height": stats.get("height", 0)
        }
    except BlockchainServiceError as e:
        raise HTTPException(status_code=500, detail=str(e))
