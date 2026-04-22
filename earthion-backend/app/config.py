"""
Earthion Backend Configuration
"""

from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Application settings."""
    
    model_config = {"extra": "ignore", "env_file": ".env", "env_file_encoding": "utf-8"}

    # Go blockchain server URL
    GO_SERVER_URL: str = "http://localhost:8333"
    
    # Supabase
    SUPABASE_URL: str = "https://cmzaaylyggsjwezsuezh.supabase.co"
    SUPABASE_KEY: str = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImNtemFheWx5Z2dzandlenN1ZXpoIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NzU2NjA4NjMsImV4cCI6MjA5MTIzNjg2M30.t81DZxqbt_OB1z3fL7s05iyVJge8ZR-kJjPNL4FTjic"
    SUPABASE_SERVICE_KEY: str = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImNtemFheWx5Z2dzandlenN1ZXpoIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NzU2NjA4NjMsImV4cCI6MjA5MTIzNjg2M30.t81DZxqbt_OB1z3fL7s05iyVJge8ZR-kJjPNL4FTjic"
    
    # API settings
    API_TITLE: str = "Earthion Blockchain API"
    API_VERSION: str = "1.0.0"
    
    # Server settings
    HOST: str = "0.0.0.0"
    PORT: int = 8000
    
    # Request timeout (seconds) - 5 minutes for mining
    REQUEST_TIMEOUT: int = 300
    
    # Validation
    MAX_TX_AMOUNT: int = 1000000000  # 1 billion max
    
    # CORS
    ALLOWED_ORIGINS: str = "http://localhost:3000,http://localhost:5173"


settings = Settings()
