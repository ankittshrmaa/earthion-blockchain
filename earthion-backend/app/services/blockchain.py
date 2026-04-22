"""
Earthion Backend - Blockchain Service
Async HTTP client for communicating with Go blockchain server.
"""

from typing import Optional
import httpx
from app.config import settings


class BlockchainServiceError(Exception):
    """Custom exception for blockchain service errors."""
    pass


class BlockchainService:
    """Async service for communicating with the Go blockchain server."""

    def __init__(self, base_url: Optional[str] = None):
        self.base_url = base_url or settings.GO_SERVER_URL
        # Configure connection pool for better performance
        self.client = httpx.AsyncClient(
            base_url=self.base_url,
            timeout=settings.REQUEST_TIMEOUT,
            limits=httpx.Limits(
                max_keepalive_connections=20,
                max_connections=100,
                keepalive_expiry=30.0
            )
        )

    async def _request(self, method: str, path: str, **kwargs) -> dict:
        """Make async HTTP request to Go server."""
        try:
            response = await self.client.request(method, path, **kwargs)
            response.raise_for_status()
            return response.json()
        except httpx.HTTPStatusError as e:
            error_data = e.response.json()
            raise BlockchainServiceError(
                error_data.get("error", f"HTTP {e.response.status_code}")
            )
        except httpx.RequestError as e:
            raise BlockchainServiceError(f"Connection failed: {str(e)}")

    async def _get(self, path: str) -> dict:
        """Make async GET request."""
        return await self._request("GET", path)

    async def _post(self, path: str, json: Optional[dict] = None) -> dict:
        """Make async POST request."""
        return await self._request("POST", path, json=json)

    # ========== Health ==========
    async def health_check(self) -> dict:
        """Check Go server health."""
        return await self._get("/health")

    # ========== Blocks ==========
    async def get_all_blocks(self) -> list[dict]:
        """Get all blocks."""
        result = await self._get("/api/blocks")
        return result.get("data", [])

    async def get_block_by_hash(self, block_hash: str) -> Optional[dict]:
        """Get block by hash."""
        result = await self._get(f"/api/blocks/{block_hash}")
        return result.get("data")

    async def get_block_by_index(self, index: int) -> Optional[dict]:
        """Get block by index."""
        result = await self._get(f"/api/blocks/index/{index}")
        return result.get("data")

    # ========== Wallet ==========
    async def get_wallet_address(self) -> dict:
        """Get wallet address."""
        result = await self._get("/api/wallet/address")
        return result.get("data", {})

    async def get_balance(self, address: str) -> int:
        """Get balance for address."""
        result = await self._get(f"/api/balance/{address}")
        data = result.get("data", {})
        return data.get("balance", 0)

    # ========== UTXO ==========
    async def get_utxo(self) -> list[dict]:
        """Get all UTXOs."""
        result = await self._get("/api/utxo")
        return result.get("data", [])

    # ========== Transactions ==========
    async def get_transaction(self, txid: str) -> Optional[dict]:
        """Get transaction by ID."""
        result = await self._get(f"/api/transaction/{txid}")
        return result.get("data")

    async def create_transaction(self, to: str, amount: int) -> dict:
        """Create new transaction."""
        result = await self._post(
            "/api/transaction",
            json={"to": to, "amount": amount}
        )
        return result.get("data", {})

    # ========== Mining ==========
    async def mine_block(self) -> dict:
        """Mine a new block."""
        result = await self._post("/api/mine")
        return result.get("data", {})

    # ========== Chain ==========
    async def validate_chain(self) -> bool:
        """Validate chain integrity."""
        result = await self._get("/api/validate")
        data = result.get("data", {})
        return data.get("valid", False)

    async def get_stats(self) -> dict:
        """Get chain statistics."""
        result = await self._get("/api/stats")
        return result.get("data", {})

    async def close(self):
        """Close async HTTP client."""
        await self.client.aclose()


# Singleton instance - lazy initialization for async context
_blockchain_service: Optional[BlockchainService] = None


def get_blockchain_service() -> BlockchainService:
    """Get or create the blockchain service singleton."""
    global _blockchain_service
    if _blockchain_service is None:
        _blockchain_service = BlockchainService()
    return _blockchain_service


async def close_blockchain_service():
    """Close the blockchain service (call on shutdown)."""
    global _blockchain_service
    if _blockchain_service is not None:
        await _blockchain_service.close()
        _blockchain_service = None