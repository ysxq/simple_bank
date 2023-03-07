package utils

// Constants for all supported currencies
const (
	USD = "USD"
	RMB = "RMB"
	EUR = "EUR"
)

// IsSupportCurrency returns true if the currency is supported
func IsSupportCurrency(currency string) bool {
	switch currency {
	case USD, RMB, EUR:
		return true
	}

	return false
}
