import httpx

# Try ANON key instead
key = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImNtemFheWx5Z2dzandlenN1ZXpoIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NzU2NjA4NjMsImV4cCI6MjA5MTIzNjg2M30.t81DZxqbt_OB1z3fL7s05iyVJge8ZR-kJjPNL4FTjic"
url = "https://cmzaaylyggsjwezsuezh.supabase.co/rest/v1/blocks?select=count"

response = httpx.get(url, headers={"apikey": key, "Authorization": f"Bearer {key}"})
print("Status:", response.status_code)
print("Response:", response.text[:200])
