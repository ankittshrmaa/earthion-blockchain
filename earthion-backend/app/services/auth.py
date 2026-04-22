"""
Earthion Backend - Authentication Service
Handles user auth via Supabase Auth.
"""

from typing import Optional
from supabase import create_client, Client
from app.config import settings


# Singleton instance
_supabase_auth: Optional[Client] = None


def get_supabase_auth() -> Client:
    """Get Supabase client for auth."""
    global _supabase_auth
    if _supabase_auth is None:
        _supabase_auth = create_client(
            settings.SUPABASE_URL,
            settings.SUPABASE_KEY
        )
    return _supabase_auth


# ========== Auth Functions ==========

def sign_up(email: str, password: str) -> dict:
    """Register a new user."""
    client = get_supabase_auth()
    return client.auth.sign_up({
        "email": email,
        "password": password
    })


def sign_in(email: str, password: str) -> dict:
    """Login user."""
    client = get_supabase_auth()
    return client.auth.sign_in_with_password({
        "email": email,
        "password": password
    })


def sign_out(access_token: str) -> dict:
    """Logout user."""
    client = get_supabase_auth()
    return client.auth.sign_out(access_token)


def get_user(access_token: str) -> dict:
    """Get current user."""
    client = get_supabase_auth()
    return client.auth.get_user(access_token)


def reset_password(email: str) -> dict:
    """Send password reset email."""
    client = get_supabase_auth()
    return client.auth.reset_password_for_email(email)


def update_password(access_token: str, new_password: str) -> dict:
    """Update user password."""
    client = get_supabase_auth()
    return client.auth.update_user(
        access_token,
        {"password": new_password}
    )


def verify_token(access_token: str) -> bool:
    """Verify if token is valid."""
    try:
        user = get_user(access_token)
        return user is not None
    except:
        return False