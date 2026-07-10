#Commodity: {
	precision: int | *5
}

// Filesystem-ish ledger directory name (parent of main.beancount).
#LedgerID: string & =~"^[A-Za-z][A-Za-z0-9_-]*$"

// One discovered ledger (injected by the host after scanning the project root).
#Ledger: {
	name: #LedgerID
	main: string // absolute path to main.beancount
}

// Map of discovered ledgers. The host injects a *closed* concrete struct:
//
//	ledgers: close({
//	  personal: {name: "personal", main: ".../personal/main.beancount"}
//	  acme:     {name: "acme",     main: ".../acme/main.beancount"}
//	})
//
// Open typing here; closedness comes from the generated fragment.
ledgers: [Name=string]: #Ledger & {
	name: Name
}

// Valid ledger name = a key of the injected ledgers map.
#LedgerName: or([for n, _ in ledgers {n}])

#LedgerRef: {
	ledger:  #LedgerName
	account: string
}

// Declarative cross-ledger reconciliation hint (not enforced by check yet).
#LedgerLink: {
	name:  string
	from:  #LedgerRef
	to:    #LedgerRef
	note?: string
}

#Config: {
	commodities: [string]: #Commodity
	operating_currency?: [...string]
	ledgers: [string]: #Ledger
	links?:  [...#LedgerLink]
}

// Default instance fields (unified with host inject + user contapila.cue).
commodities: [string]: #Commodity
links?:      [...#LedgerLink]
