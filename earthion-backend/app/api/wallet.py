"""Wallet API endpoints."""

from fastapi import APIRouter, HTTPException, Request
from slowapi import Limiter
from slowapi.util import get_remote_address
from slowapi.errors import RateLimitExceeded

from app.models import WalletAddressResponse, BalanceResponse, CreateTransactionRequest, TransactionResponse
from app.services.blockchain import get_blockchain_service, BlockchainServiceError

router = APIRouter()

# Rate limiter
limiter = Limiter(key_func=get_remote_address)


@router.get("/address", response_model=WalletAddressResponse)
async def get_wallet_address():
    """
    Get wallet address.
    """
    try:
        service = get_blockchain_service()
        data = await service.get_wallet_address()
        return WalletAddressResponse(**data)
    except BlockchainServiceError as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/balance", response_model=BalanceResponse)
async def get_wallet_balance():
    """
    Get wallet balance.
    """
    try:
        service = get_blockchain_service()
        # Get wallet address first
        wallet_data = await service.get_wallet_address()
        raw_address = wallet_data.get("raw")
        
        if not raw_address:
            raise HTTPException(status_code=500, detail="Could not get wallet address")
        
        balance = await service.get_balance(raw_address)
        return BalanceResponse(balance=balance)
    except BlockchainServiceError as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.post("/send", response_model=TransactionResponse)
@limiter.limit("20/minute")  # Limit sending to 20 per minute
async def send_coins(request: Request, req: CreateTransactionRequest):
    """
    Send coins to another address.
    
    Rate limited to 10 requests per minute.
    """
    try:
        service = get_blockchain_service()
        tx_data = await service.create_transaction(
            to=req.to,
            amount=req.amount
        )
        return TransactionResponse(transaction=tx_data)
    except RateLimitExceeded:
        raise HTTPException(status_code=429, detail="Rate limit exceeded. Try again later.")
    except BlockchainServiceError as e:
        raise HTTPException(status_code=400, detail=str(e))
