"""
Earthion Backend - Pydantic Models
Data models for API requests and responses.
"""

from typing import Optional
from pydantic import BaseModel, Field, field_validator



# Block Models
class TXInputModel(BaseModel):
    """Transaction input model."""
    txid: str
    out_index: int = Field(alias="outIndex")
    signature: str
    pub_key: str = Field(alias="pubKey")
    
    class Config:
        populate_by_name = True


class TXOutputModel(BaseModel):
    """Transaction output model."""
    value: int
    pub_key: str = Field(alias="pubKey")
    
    class Config:
        populate_by_name = True


class TransactionModel(BaseModel):
    """Transaction model."""
    id: str
    inputs: list[TXInputModel]
    outputs: list[TXOutputModel]
    
    class Config:
        populate_by_name = True


class BlockModel(BaseModel):
    """Block model."""
    index: int
    timestamp: int
    prev_hash: str = Field(alias="prevHash")
    merkle_root: str = Field(alias="merkleRoot")
    hash: str
    nonce: int
    difficulty: int
    transactions: list[TransactionModel] = []
    tx_count: int = Field(default=0, alias="txCount")
    
    class Config:
        populate_by_name = True
        populate_by_alias = True
    
    def __init__(self, **data):
        # Handle both 'transactions' and compute txCount
        if 'transactions' in data:
            txns = data['transactions']
            data['transactions'] = txns
            data['tx_count'] = len(txns) if txns else 0
        elif 'txCount' in data:
            data['tx_count'] = data['txCount']
        # Set default if not provided
        if 'tx_count' not in data:
            data['tx_count'] = 0
        super().__init__(**data)
    
    def to_camelcase_dict(self):
        """Convert to dict with camelCase keys for JSON response."""
        d = self.model_dump(by_alias=True, exclude={'transactions'})
        d['txCount'] = self.tx_count
        # Include transaction IDs only for brevity
        d['transactions'] = [{'id': tx.id} for tx in self.transactions] if self.transactions else []
        return d


class BlockListResponse(BaseModel):
    """Response model for block list."""
    blocks: list[BlockModel]
    count: int


# Wallet Models
class WalletAddressResponse(BaseModel):
    """Wallet address response."""
    address: str  # Base58 encoded
    raw: str     # 20-byte hex


class BalanceResponse(BaseModel):
    """Balance response."""
    balance: int


# Transaction Models
class CreateTransactionRequest(BaseModel):
    """Request model for creating a transaction."""
    to: str = Field(..., min_length=40, max_length=40, description="Recipient address (40 hex chars)")
    amount: int = Field(..., gt=0, le=1000000000, description="Amount to send")
    
    @field_validator('to')
    @classmethod
    def validate_address(cls, v: str) -> str:
        if not all(c in '0123456789abcdefABCDEF' for c in v):
            raise ValueError('Address must be valid hex')
        return v.lower()


class TransactionResponse(BaseModel):
    """Transaction response."""
    transaction: TransactionModel


# Mining Models
class MineBlockResponse(BaseModel):
    """Response model for mining a block."""
    block: BlockModel
    reward: int


# Chain Models
class ValidateResponse(BaseModel):
    """Chain validation response."""
    valid: bool


# Stats Models - Support both camelCase and snake_case
class ChainStatsResponse(BaseModel):
    """Chain statistics response."""
    height: int
    difficulty: int
    total_work: Optional[int] = Field(default=None, alias="totalWork")
    total_mined: Optional[int] = Field(default=None, alias="totalMined")
    current_reward: Optional[int] = Field(default=None, alias="currentReward")
    max_supply: Optional[int] = Field(default=None, alias="maxSupply")
    
    class Config:
        populate_by_name = True


# UTXO Models
class UTXOModel(BaseModel):
    """UTXO model."""
    key: str
    value: TXOutputModel


class UTXOResponse(BaseModel):
    """UTXO response."""
    utxos: list[UTXOModel]
    count: int


# Generic Response
class APIResponse(BaseModel):
    """Generic API response."""
    success: bool
    data: Optional[dict] = None
    error: Optional[str] = None