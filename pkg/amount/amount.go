package amount

import (
	"fmt"
	"math/big"
)

type Amount struct {
	Num       *big.Rat
	Commodity string
}

func New(num *big.Rat, commodity string) Amount {
	return Amount{
		Num:       new(big.Rat).Set(num),
		Commodity: commodity,
	}
}

func (a Amount) Clone() Amount {
	return New(a.Num, a.Commodity)
}

func (a Amount) Neg() Amount {
	return Amount{
		Num:       new(big.Rat).Neg(a.Num),
		Commodity: a.Commodity,
	}
}

func (a Amount) Abs() Amount {
	return Amount{
		Num:       new(big.Rat).Abs(a.Num),
		Commodity: a.Commodity,
	}
}

func (a Amount) Add(other Amount) (Amount, error) {
	if a.Commodity != other.Commodity {
		return Amount{}, fmt.Errorf("commodity mismatch in addition: %s != %s", a.Commodity, other.Commodity)
	}
	return Amount{
		Num:       new(big.Rat).Add(a.Num, other.Num),
		Commodity: a.Commodity,
	}, nil
}

func (a Amount) Sub(other Amount) (Amount, error) {
	if a.Commodity != other.Commodity {
		return Amount{}, fmt.Errorf("commodity mismatch in subtraction: %s != %s", a.Commodity, other.Commodity)
	}
	return Amount{
		Num:       new(big.Rat).Sub(a.Num, other.Num),
		Commodity: a.Commodity,
	}, nil
}

func (a Amount) Mul(r *big.Rat) Amount {
	return Amount{
		Num:       new(big.Rat).Mul(a.Num, r),
		Commodity: a.Commodity,
	}
}

func (a Amount) Div(r *big.Rat) Amount {
	return Amount{
		Num:       new(big.Rat).Quo(a.Num, r),
		Commodity: a.Commodity,
	}
}

func (a Amount) Zero() bool {
	return a.Num == nil || a.Num.Sign() == 0
}

func (a Amount) Cmp(other Amount) (int, error) {
	if a.Commodity != other.Commodity {
		return 0, fmt.Errorf("commodity mismatch in comparison: %s != %s", a.Commodity, other.Commodity)
	}
	return a.Num.Cmp(other.Num), nil
}

func (a Amount) EqualWithTolerance(other Amount, tolerance *big.Rat) bool {
	if a.Commodity != other.Commodity {
		return false
	}
	diff := new(big.Rat).Sub(a.Num, other.Num)
	diff.Abs(diff)
	return diff.Cmp(tolerance) <= 0
}

func (a Amount) String() string {
	if a.Num == nil {
		return "0 " + a.Commodity
	}
	return a.Num.FloatString(10) + " " + a.Commodity
}
