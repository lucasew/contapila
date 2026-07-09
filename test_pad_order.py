import datetime
from beancount import loader
from beancount.core import data, inventory

content = """
2023-01-01 open Assets:Cash
2023-01-01 open Equity:Opening-Balances

2023-01-02 pad Assets:Cash Equity:Opening-Balances
2023-01-03 balance Assets:Cash 100 USD
2023-01-01 * "Initial"
  Assets:Cash  50 USD
  Equity:Opening-Balances
"""

entries, errors, options = loader.load_string(content)
print(f"Errors: {len(errors)}")
for e in errors:
    print(e.message)

for entry in entries:
    if isinstance(entry, data.Transaction):
        print(f"Transaction on {entry.date} ({entry.narration}):")
        for p in entry.postings:
            print(f"  {p.account} {p.units}")
