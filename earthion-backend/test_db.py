from supabase import create_client

key = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImNtemFheWx5Z2dzandlenN1ZXpoIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NzU2NjA4NjMsImV4cCI6MjA5MTIzNjg2M30.t81DZxqbt_OB1z3fL7s05iyVJge8ZR-kJjPNL4FTjic"
url = "https://cmzaaylyggsjwezsuezh.supabase.co"

client = create_client(url, key)
result = client.table("user_wallets").select("*").execute()
print("Count:", len(result.data))
for row in result.data:
    print(row)