from beancount import loader
content = """
2023-01-01 open Assets:Cash
2023-01-01 open Equity:Opening-Balances
2023-01-01 pad Assets:Cash Equity:Opening-Balances
2023-01-01 balance Assets:Cash 100 USD
"""
entries, errors, options = loader.load_string(content)
print(f"Errors: {len(errors)}")
for e in errors:
    print(e.message)
