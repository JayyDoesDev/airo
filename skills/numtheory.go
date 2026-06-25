package skills

import (
	"fmt"
	"math"
	"strings"
)

type NumberTheoryConfig struct {
	Operation string    `json:"operation"`
	Numbers   []int64   `json:"numbers"`
	RangeEnd  int64     `json:"range_end,omitempty"`
}

type NumberTheoryResult struct {
	Operation string
	Input     []int64
	Output    string
	Extra     map[string]interface{}
}

func RunNumberTheory(cfg NumberTheoryConfig) (NumberTheoryResult, error) {
	if len(cfg.Numbers) == 0 {
		return NumberTheoryResult{}, fmt.Errorf("no numbers provided")
	}

	result := NumberTheoryResult{Operation: cfg.Operation, Input: cfg.Numbers, Extra: map[string]interface{}{}}

	switch cfg.Operation {
	case "factorize":
		var parts []string
		for _, n := range cfg.Numbers {
			if n < 1 {
				return NumberTheoryResult{}, fmt.Errorf("factorization requires positive integers")
			}
			if n > 1e15 {
				return NumberTheoryResult{}, fmt.Errorf("number too large (max 10^15)")
			}
			factors := primeFactors(n)
			parts = append(parts, fmt.Sprintf("%d = %s", n, formatFactors(factors)))
		}
		result.Output = strings.Join(parts, "\n")

	case "is_prime":
		var parts []string
		for _, n := range cfg.Numbers {
			if isPrime(n) {
				parts = append(parts, fmt.Sprintf("%d is **prime**", n))
			} else {
				parts = append(parts, fmt.Sprintf("%d is **not prime** (factors: %s)", n, formatFactors(primeFactors(n))))
			}
		}
		result.Output = strings.Join(parts, "\n")

	case "gcd":
		if len(cfg.Numbers) < 2 {
			return NumberTheoryResult{}, fmt.Errorf("gcd requires at least 2 numbers")
		}
		g := cfg.Numbers[0]
		for _, n := range cfg.Numbers[1:] {
			g = gcd(g, n)
		}
		result.Output = fmt.Sprintf("GCD(%s) = **%d**", joinNums(cfg.Numbers), g)

	case "lcm":
		if len(cfg.Numbers) < 2 {
			return NumberTheoryResult{}, fmt.Errorf("lcm requires at least 2 numbers")
		}
		l := cfg.Numbers[0]
		for _, n := range cfg.Numbers[1:] {
			l = lcm(l, n)
			if l > 1e15 {
				return NumberTheoryResult{}, fmt.Errorf("LCM too large to compute")
			}
		}
		result.Output = fmt.Sprintf("LCM(%s) = **%d**", joinNums(cfg.Numbers), l)

	case "primes_in_range":
		start := cfg.Numbers[0]
		end := cfg.RangeEnd
		if end == 0 && len(cfg.Numbers) >= 2 {
			end = cfg.Numbers[1]
		}
		if end-start > 100000 {
			return NumberTheoryResult{}, fmt.Errorf("range too large (max 100000)")
		}
		primes := sieve(start, end)
		if len(primes) == 0 {
			result.Output = fmt.Sprintf("no primes between %d and %d", start, end)
		} else if len(primes) > 50 {
			result.Output = fmt.Sprintf("found **%d primes** between %d and %d\nFirst 10: %s\nLast 10: %s",
				len(primes), start, end,
				joinNums(primes[:10]),
				joinNums(primes[len(primes)-10:]))
		} else {
			result.Output = fmt.Sprintf("primes between %d and %d (%d total):\n%s", start, end, len(primes), joinNums(primes))
		}

	case "fibonacci":
		n := cfg.Numbers[0]
		if n < 0 || n > 80 {
			return NumberTheoryResult{}, fmt.Errorf("n must be between 0 and 80")
		}
		seq := fibonacci(int(n))
		result.Output = fmt.Sprintf("Fibonacci sequence (first %d terms):\n%s", n, joinNums(seq))

	case "divisors":
		n := cfg.Numbers[0]
		if n < 1 || n > 1e9 {
			return NumberTheoryResult{}, fmt.Errorf("n must be between 1 and 10^9")
		}
		divs := divisors(n)
		result.Output = fmt.Sprintf("divisors of %d (%d total): %s\nsum of divisors: %d", n, len(divs), joinNums(divs), sumSlice(divs))

	default:
		return NumberTheoryResult{}, fmt.Errorf("unknown operation %q. supported: factorize, is_prime, gcd, lcm, primes_in_range, fibonacci, divisors", cfg.Operation)
	}

	return result, nil
}

func primeFactors(n int64) []int64 {
	var factors []int64
	for n%2 == 0 {
		factors = append(factors, 2)
		n /= 2
	}
	for i := int64(3); i <= int64(math.Sqrt(float64(n))); i += 2 {
		for n%i == 0 {
			factors = append(factors, i)
			n /= i
		}
	}
	if n > 2 {
		factors = append(factors, n)
	}
	return factors
}

func formatFactors(factors []int64) string {
	if len(factors) == 0 {
		return "1"
	}
	counts := map[int64]int{}
	var order []int64
	for _, f := range factors {
		if counts[f] == 0 {
			order = append(order, f)
		}
		counts[f]++
	}
	var parts []string
	for _, f := range order {
		if counts[f] == 1 {
			parts = append(parts, fmt.Sprintf("%d", f))
		} else {
			parts = append(parts, fmt.Sprintf("%d^%d", f, counts[f]))
		}
	}
	return strings.Join(parts, " × ")
}

func isPrime(n int64) bool {
	if n < 2 {
		return false
	}
	if n == 2 {
		return true
	}
	if n%2 == 0 {
		return false
	}
	for i := int64(3); i <= int64(math.Sqrt(float64(n))); i += 2 {
		if n%i == 0 {
			return false
		}
	}
	return true
}

func gcd(a, b int64) int64 {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func lcm(a, b int64) int64 {
	return a / gcd(a, b) * b
}

func sieve(start, end int64) []int64 {
	if end < 2 {
		return nil
	}
	if start < 2 {
		start = 2
	}
	size := end - start + 1
	composite := make([]bool, size)
	for i := int64(2); i*i <= end; i++ {
		first := ((start + i - 1) / i) * i
		if first == i {
			first += i
		}
		for j := first; j <= end; j += i {
			composite[j-start] = true
		}
	}
	var primes []int64
	for i := int64(0); i < size; i++ {
		if !composite[i] {
			primes = append(primes, start+i)
		}
	}
	return primes
}

func fibonacci(n int) []int64 {
	if n == 0 {
		return []int64{}
	}
	seq := make([]int64, n)
	seq[0] = 0
	if n > 1 {
		seq[1] = 1
	}
	for i := 2; i < n; i++ {
		seq[i] = seq[i-1] + seq[i-2]
	}
	return seq
}

func divisors(n int64) []int64 {
	var divs []int64
	for i := int64(1); i*i <= n; i++ {
		if n%i == 0 {
			divs = append(divs, i)
			if i != n/i {
				divs = append(divs, n/i)
			}
		}
	}
	sorted := make([]int64, len(divs))
	copy(sorted, divs)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	return sorted
}

func joinNums(nums []int64) string {
	parts := make([]string, len(nums))
	for i, n := range nums {
		parts[i] = fmt.Sprintf("%d", n)
	}
	return strings.Join(parts, ", ")
}

func sumSlice(nums []int64) int64 {
	var s int64
	for _, n := range nums {
		s += n
	}
	return s
}
