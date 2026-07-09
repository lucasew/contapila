operating_currency: [...string]
commodities: {
	[string]: {
		precision: int | *5
	}
}
accounts: {
	[string]: {
		opened: bool | *false
		closed: bool | *false
		currencies: [...string]
	}
}
