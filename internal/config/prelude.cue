#Commodity: {
    precision: int | *5
}

#Config: {
    commodities: [string]: #Commodity
    operating_currency?: [...string]
}

// Default instance
commodities: [string]: #Commodity
