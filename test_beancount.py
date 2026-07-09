import datetime
from beancount import loader
from beancount.core import data, inventory

content = """
2023-01-01 open Assets:Cash
2023-01-01 open Equity:Opening-Balances

2023-01-01 * "Initial"
  Assets:Cash  100 USD
  Equity:Opening-Balances

2023-01-02 balance Assets:Cash 100 USD
"""

entries, errors, options = loader.load_string(content)
print(f"Errors with balance on 2023-01-02 (after txn on 01-01): {len(errors)}")
for e in errors:
    print(e.message)

content2 = """
2023-01-01 open Assets:Cash
2023-01-01 open Equity:Opening-Balances

2023-01-01 * "Initial"
  Assets:Cash  100 USD
  Equity:Opening-Balances

2023-01-01 balance Assets:Cash 100 USD
"""
entries, errors, options = loader.load_string(content2)
print(f"Errors with balance on 2023-01-01 (same day as txn): {len(errors)}")
for e in errors:
    print(e.message)

content3 = """
2023-01-01 open Assets:Cash
2023-01-01 open Equity:Opening-Balances

2023-01-01 balance Assets:Cash 0 USD
2023-01-01 * "Initial"
  Assets:Cash  100 USD
  Equity:Opening-Balances
"""
entries, errors, options = loader.load_string(content3)
print(f"Errors with balance on 2023-01-01 (before txn in file): {len(errors)}")
for e in errors:
    print(e.message)
