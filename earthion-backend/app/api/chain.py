"""Chain API endpoints."""

from fastapi import APIRouter, HTTPException

from app.models import ValidateResponse
from app.services.blockchain import get_blockchain_service, BlockchainServiceError

router = APIRouter()


@router.get("/validate", response_model=ValidateResponse)
async def validate_chain():
    """
    Validate blockchain integrity.
    
    Checks:
    - Genesis block is correct
    - All blocks linked properly
    - PoW is valid
    - Merkle roots match
    - No double-spends
    - All signatures valid
    """
    try:
        service = get_blockchain_service()
        valid = await service.validate_chain()
        return ValidateResponse(valid=valid)
    except BlockchainServiceError as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/height")
async def get_chain_height():
    """
    Get current chain height.
    """
    try:
        service = get_blockchain_service()
        stats = await service.get_stats()
        return {
            "height": stats.get("height", 0)
        }
    except BlockchainServiceError as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/utxo")
async def get_utxo():
    """
    Get all unspent transaction outputs.
    """
    try:
        service = get_blockchain_service()
        utxos = await service.get_utxo()
        return {
            "utxos": utxos,
            "count": len(utxos)
        }
    except BlockchainServiceError as e:
        raise HTTPException(status_code=500, detail=str(e))
