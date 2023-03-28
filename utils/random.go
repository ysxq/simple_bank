package utils

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

var random *rand.Rand

func init() {
	// rand.Seed(time.Now().UnixNano())
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// return a random integer between min and max
func RandomInt(min, max int64) int64 {
	return min + random.Int63n(max-min+1)
}

// return a random string of length n
func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[random.Intn(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}

// return a random owner name
func RandomOwner() string {
	return RandomString(6)
}

// return a random amount of money
func RandomMoney() int64 {
	return RandomInt(0, 1000)
}

// return a random currency code
func RandomCurrency() string {
	currencies := []string{USD, RMB, EUR} // 美元、人民币、欧元
	n := len(currencies)
	return currencies[random.Intn(n)]
}

func RandomEmail() string {
	return fmt.Sprintf("%v@email.com", RandomString(6))
}
