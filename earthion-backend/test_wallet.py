import sys
sys.path.insert(0, '.')
from app.services import wallet_manager

# Try creating a wallet
result = wallet_manager.create_user_wallet("test-user-456", "test2@test.com", "John Smith")
print("Result:", result)

# Check if it exists
wallet = wallet_manager.get_user_wallet("test-user-456")
print("Wallet:", wallet)