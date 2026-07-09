import datetime
from beancount import loader
from beancount.core import data, inventory

content = """
2023-01-01 open Assets:Cash
2023-01-01 open Equity:Opening-Balances

2023-01-01 pad Assets:Cash Equity:Opening-Balances
2023-01-02 balance Assets:Cash 100 USD
"""

entries, errors, options = loader.load_string(content)
print(f"Errors with pad: {len(errors)}")
for e in errors:
    print(e.message)

for entry in entries:
    if isinstance(entry, data.Transaction):
        print(f"Generated transaction on {entry.date}:")
        for p in entry.postings:
            print(f"  {p.account} {p.units}")
