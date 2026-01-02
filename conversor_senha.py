import urllib.parse
senha = "SUA_SENHA_ORIGINAL"
senha_encoded = urllib.parse.quote(senha, safe='')
print("\n=== SENHA ENCODED ===")
print(senha_encoded)
print("\n=== COPIE A LINHA ABAIXO ===")
print(f'export DATABASE_URL="postgres://postgres:{senha_encoded}@rds-finance-tracker.cqh40koaapaj.us-east-1.rds.amazonaws.com:5432/financetracker?sslmode=require"')