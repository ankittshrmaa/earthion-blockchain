import requests

BASE_URL = "http://localhost:8000/api/auth"

# Test signup
print("=== Testing Signup ===")
resp = requests.post(f"{BASE_URL}/signup", json={
    "email": "testuser123@test.com",
    "password": "testpass123"
})
print("Status:", resp.status_code)
print("Response:", resp.text[:300])

# Test signin
print("\n=== Testing Signin ===")
resp = requests.post(f"{BASE_URL}/signin", json={
    "email": "testuser123@test.com",
    "password": "testpass123"
})
print("Status:", resp.status_code)
print("Response:", resp.text[:500])
