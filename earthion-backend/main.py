"""
Earthion Backend - FastAPI Application
Main entry point for the blockchain API server.
"""

import os
import logging
from contextlib import asynccontextmanager
from fastapi import FastAPI, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from slowapi import Limiter
from slowapi.util import get_remote_address
from slowapi.errors import RateLimitExceeded
from slowapi import _rate_limit_exceeded_handler
import time

from app.api import blocks, wallet, mining, chain, stats, auth
from app.config import settings
from app.services.blockchain import get_blockchain_service, close_blockchain_service, BlockchainServiceError
from app.services.sync import start_sync_loop, test_connection
import asyncio


# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger("earthion-backend")

# Rate limiter
limiter = Limiter(key_func=get_remote_address)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan handler."""
    # Startup
    logger.info(f"Starting Earthion Backend...")
    logger.info(f"Go Server URL: {settings.GO_SERVER_URL}")
    
    # Verify Go server connectivity
    service = get_blockchain_service()
    try:
        await service.health_check()
        logger.info("Go server connectivity: OK")
    except BlockchainServiceError as e:
        logger.warning(f"Go server connectivity: FAILED - {e}")
    
    # Test Supabase connection
    supabase_ok = await test_connection()
    if supabase_ok:
        logger.info("Starting blockchain sync to Supabase...")
        # Start background sync (non-blocking)
        asyncio.create_task(start_sync_loop(interval=30))
    
    yield
    # Shutdown
    logger.info("Shutting down Earthion Backend...")
    await close_blockchain_service()
    logger.info("Blockchain service closed")


app = FastAPI(
    title="Earthion Blockchain API",
    description="REST API for Earthion blockchain operations",
    version="1.0.0",
    lifespan=lifespan,
)

# Rate limiting
app.state.limiter = limiter
app.add_exception_handler(RateLimitExceeded, _rate_limit_exceeded_handler)


@app.middleware("http")
async def log_requests(request: Request, call_next):
    """Log all requests."""
    start_time = time.time()
    
    response = await call_next(request)
    
    duration = time.time() - start_time
    logger.info(
        f"{request.method} {request.url.path} - "
        f"Status: {response.status_code} - "
        f"Duration: {duration:.3f}s"
    )
    
    return response


# CORS middleware - configure properly for production
# In production, replace with specific origins
allowed_origins = os.getenv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173").split(",")

app.add_middleware(
    CORSMiddleware,
    allow_origins=allowed_origins,  # Configure appropriately for production
    allow_credentials=True,
    allow_methods=["GET", "POST", "PUT", "DELETE"],
    allow_headers=["*"],
)

# Include routers
app.include_router(blocks.router, prefix="/api/blocks", tags=["Blocks"])
app.include_router(wallet.router, prefix="/api/wallet", tags=["Wallet"])
app.include_router(mining.router, prefix="/api/mining", tags=["Mining"])
app.include_router(chain.router, prefix="/api/chain", tags=["Chain"])
app.include_router(stats.router, prefix="/api", tags=["Stats"])
app.include_router(auth.router, prefix="/api/auth", tags=["Auth"])


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    service = get_blockchain_service()
    try:
        # Probe Go server
        go_health = await service.health_check()
        return {
            "status": "healthy",
            "service": "earthion-backend",
            "go_server": "connected",
            "go_health": go_health
        }
    except BlockchainServiceError as e:
        return {
            "status": "degraded",
            "service": "earthion-backend",
            "go_server": "disconnected",
            "error": str(e)
        }


@app.get("/")
async def root():
    """Root endpoint."""
    return {
        "service": "Earthion Blockchain API",
        "version": "1.0.0",
        "docs": "/docs",
    }


if __name__ == "__main__":
    import uvicorn
    port = int(os.getenv("PORT", "8000"))
    uvicorn.run(app, host="0.0.0.0", port=port)
