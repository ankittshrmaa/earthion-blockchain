"""
Earthion Backend - Auth API
User registration and login endpoints.
"""

from fastapi import APIRouter, HTTPException, Header
from pydantic import BaseModel
from typing import Optional
from app.services import auth as auth_service
from app.services import wallet_manager


router = APIRouter()


# Request/Response models
class SignUpRequest(BaseModel):
    email: str
    password: str
    name: Optional[str] = None


class SignInRequest(BaseModel):
    email: str
    password: str


class AuthResponse(BaseModel):
    success: bool
    message: str
    user: Optional[dict] = None
    username: Optional[str] = None
    access_token: Optional[str] = None


@router.post("/signup", response_model=AuthResponse)
async def sign_up(request: SignUpRequest):
    """Register a new user."""
    try:
        result = auth_service.sign_up(request.email, request.password)
        
        if result.user:
            # Create wallet with user name
            user_id = result.user.id
            # Get username from name or email prefix
            username = request.name or request.email.split('@')[0]
            wallet = wallet_manager.create_user_wallet(user_id, request.email, username)
            
            return AuthResponse(
                success=True,
                message="Account created successfully!",
                user={"id": user_id, "email": result.user.email, "username": username},
                username=username
            )
        else:
            return AuthResponse(
                success=True,
                message="Account created! Check email to verify."
            )
            
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.post("/signin", response_model=AuthResponse)
async def sign_in(request: SignInRequest):
    """Login user - creates wallet if doesn't exist."""
    try:
        result = auth_service.sign_in(request.email, request.password)
        
        if hasattr(result, 'session') and result.session:
            user_id = result.user.id
            email = result.user.email
            
            # Create wallet if doesn't exist
            wallet = wallet_manager.get_user_wallet(user_id)
            if not wallet:
                wallet = wallet_manager.create_user_wallet(user_id, email)
            
            # Get username from email prefix
            username = email.split('@')[0]
            
            return AuthResponse(
                success=True,
                message="Login successful",
                user={
                    "id": user_id, 
                    "email": email,
                    "username": username,
                    "address": wallet.get("address") if wallet else None
                },
                access_token=result.session.access_token
            )
        elif hasattr(result, 'user') and result.user:
            raise HTTPException(
                status_code=401, 
                detail="Please verify your email first. Check your inbox for verification link."
            )
        else:
            return AuthResponse(
                success=False,
                message="Invalid credentials"
            )
            
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=401, detail="Invalid email or password")


@router.post("/signout")
async def sign_out(authorization: Optional[str] = Header(None)):
    """Logout user."""
    if not authorization or not authorization.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="Missing token")
    
    token = authorization.replace("Bearer ", "")
    
    try:
        auth_service.sign_out(token)
        return {"success": True, "message": "Logged out"}
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.get("/me")
async def get_me(authorization: Optional[str] = Header(None)):
    """Get current user with wallet."""
    if not authorization or not authorization.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="Missing token")
    
    token = authorization.replace("Bearer ", "")
    
    try:
        user = auth_service.get_user(token)
        user_id = user.user.id
        
        # Get user's wallet
        wallet = wallet_manager.get_user_wallet(user_id)
        
        # Get username from user metadata or use email prefix
        username = user.user.email.split('@')[0] if user.user.email else "user"
        
        return {
            "success": True,
            "user": {
                "id": user.user.id,
                "email": user.user.email,
                "username": username,
                "address": wallet.get("address") if wallet else None,
                "balance": wallet.get("balance", 0) if wallet else 0
            }
        }
    except Exception as e:
        raise HTTPException(status_code=401, detail="Invalid token")


@router.post("/reset-password")
async def reset_password(email: str):
    """Send password reset email."""
    try:
        auth_service.reset_password(email)
        return {"success": True, "message": "Password reset email sent"}
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.post("/update-password")
async def update_password(
    new_password: str,
    authorization: Optional[str] = Header(None)
):
    """Update password."""
    if not authorization or not authorization.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="Missing token")
    
    token = authorization.replace("Bearer ", "")
    
    try:
        auth_service.update_password(token, new_password)
        return {"success": True, "message": "Password updated"}
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.get("/wallet")
async def get_user_wallet(authorization: Optional[str] = Header(None)):
    """Get current user's wallet info."""
    if not authorization or not authorization.startswith("Bearer "):
        raise HTTPException(status_code=401, detail="Missing token")
    
    token = authorization.replace("Bearer ", "")
    
    try:
        user = auth_service.get_user(token)
        wallet = wallet_manager.get_user_wallet(user.user.id)
        
        if not wallet:
            # Create wallet if doesn't exist
            wallet = wallet_manager.create_user_wallet(user.user.id, user.user.email)
        
        return {
            "success": True,
            "wallet": wallet
        }
    except Exception as e:
        raise HTTPException(status_code=401, detail="Invalid token")
