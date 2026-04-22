"""
Earthion Backend Integration Tests

Tests require:
1. Go server running on localhost:8333
2. Python backend can connect to Go server

Run with: pytest tests/test_integration.py -v
"""

import pytest
import httpx
from app.config import settings


# Test configuration
GO_SERVER_URL = settings.GO_SERVER_URL
BACKEND_URL = "http://localhost:8000"


@pytest.fixture(scope="module")
def go_server():
    """Ensure Go server is running."""
    # Note: Go server should be started separately
    # This fixture just provides the URL
    yield GO_SERVER_URL


@pytest.fixture(scope="module")
def backend_server():
    """Ensure backend server is running."""
    # Note: Backend should be started separately
    # This fixture just provides the URL
    yield BACKEND_URL


class TestGoServer:
    """Tests for Go blockchain server."""

    def test_health_check(self, go_server):
        """Test Go server health."""
        response = httpx.get(f"{go_server}/health")
        assert response.status_code == 200
        data = response.json()
        assert data["success"] is True
        assert data["data"]["status"] == "healthy"

    def test_get_wallet_address(self, go_server):
        """Test getting wallet address."""
        response = httpx.get(f"{go_server}/api/wallet/address")
        assert response.status_code == 200
        data = response.json()
        assert data["success"] is True
        assert "address" in data["data"]
        assert "raw" in data["data"]

    def test_get_stats(self, go_server):
        """Test getting chain stats."""
        response = httpx.get(f"{go_server}/api/stats")
        assert response.status_code == 200
        data = response.json()
        assert data["success"] is True
        assert "height" in data["data"]
        assert "difficulty" in data["data"]

    def test_get_balance(self, go_server):
        """Test getting balance."""
        # Get wallet address first
        response = httpx.get(f"{go_server}/api/wallet/address")
        raw_address = response.json()["data"]["raw"]
        
        # Get balance
        response = httpx.get(f"{go_server}/api/balance/{raw_address}")
        assert response.status_code == 200
        data = response.json()
        assert data["success"] is True
        assert "balance" in data["data"]

    def test_mine_block(self, go_server):
        """Test mining a block."""
        response = httpx.post(f"{go_server}/api/mine")
        assert response.status_code == 200
        data = response.json()
        assert data["success"] is True
        assert "index" in data["data"]
        assert "hash" in data["data"]

    def test_validate_chain(self, go_server):
        """Test chain validation."""
        response = httpx.get(f"{go_server}/api/validate")
        assert response.status_code == 200
        data = response.json()
        assert data["success"] is True
        assert "valid" in data["data"]

    def test_get_blocks(self, go_server):
        """Test getting all blocks."""
        response = httpx.get(f"{go_server}/api/blocks")
        assert response.status_code == 200
        data = response.json()
        assert data["success"] is True
        assert isinstance(data["data"], list)


class TestBackendAPI:
    """Tests for Python FastAPI backend."""

    def test_backend_health(self, backend_server):
        """Test backend health."""
        response = httpx.get(f"{backend_server}/health")
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "healthy"

    def test_backend_root(self, backend_server):
        """Test backend root."""
        response = httpx.get(f"{backend_server}/")
        assert response.status_code == 200
        data = response.json()
        assert "service" in data

    def test_backend_blocks_endpoint(self, backend_server):
        """Test blocks endpoint exists."""
        response = httpx.get(f"{backend_server}/api/blocks")
        # May fail if Go server not running, but endpoint should exist
        assert response.status_code in [200, 500]

    def test_backend_wallet_endpoint(self, backend_server):
        """Test wallet endpoint exists."""
        response = httpx.get(f"{backend_server}/api/wallet/address")
        # May fail if Go server not running, but endpoint should exist
        assert response.status_code in [200, 500]


class TestIntegration:
    """End-to-end integration tests."""

    def test_full_workflow(self, go_server):
        """Test complete workflow: check balance, mine, verify."""
        # 1. Get initial balance
        response = httpx.get(f"{go_server}/api/wallet/address")
        raw_address = response.json()["data"]["raw"]
        
        response = httpx.get(f"{go_server}/api/balance/{raw_address}")
        initial_balance = response.json()["data"]["balance"]
        
        # 2. Mine a block
        response = httpx.post(f"{go_server}/api/mine")
        assert response.status_code == 200
        
        # 3. Get new balance
        response = httpx.get(f"{go_server}/api/balance/{raw_address}")
        new_balance = response.json()["data"]["balance"]
        
        # 4. Verify reward added
        assert new_balance > initial_balance
        
        # 5. Validate chain
        response = httpx.get(f"{go_server}/api/validate")
        assert response.json()["data"]["valid"] is True
