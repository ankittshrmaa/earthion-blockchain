import requests

BASE_URL = "http://localhost:8000/api/auth"

# Create NEW user - fresh email
import time
email = f"newuser{int(time.time())}@test.com"

print("=== Signup ===")
resp = requests.post(f"{BASE_URL}/signup", json={
    "email": email,
    "password": "testpass123"
})
print("Status:", resp.status_code)
print("Response:", resp.text[:300])

print("\n=== Signin (immediately) ===")
resp = requests.post(f"{BASE_URL}/signin", json={
    "email": email,
    "password": "testpass123"
})
print("Status:", resp.status_code)
print("Response:", resp.text[:300])
