package CodeGenerator

import (
	"fmt"
	"math/rand"
	"strconv"
)

type ConstNumber struct {
	Value      uint64
	BitWidth   int
	Signedness bool // true -> signed; false -> unsigned
}

// ToVerilogLiteral returns the Verilog literal representation of the constant
func (c ConstNumber) ToVerilogLiteral() string {
	sign := ""
	if c.Signedness {
		sign = "s"
	}

	// Mask the value to truncate to BitWidth bits
	mask := uint64((1 << c.BitWidth) - 1)
	truncated := c.Value & mask

	// Convert to binary string
	binary := strconv.FormatUint(truncated, 2)

	// Pad with leading zeros if necessary
	for len(binary) < c.BitWidth {
		binary = "0" + binary
	}

	return fmt.Sprintf("%d'%sb%s", c.BitWidth, sign, binary)
}

// RandomConstNumber generates a random ConstNumber with 1–34 bit width and non-zero value
func RandomConstNumber() ConstNumber {
	bitWidth := rand.Intn(34) + 1 // 1 to 34 bits
	signed := false               // true for signed, false for unsigned

	var value uint64
	if signed {
		max := int64(1) << (bitWidth - 1)
		for {
			rangeVal := rand.Int63n(2 * max)
			v := int64(rangeVal - max)
			if v != 0 {
				value = uint64(v)
				break
			}
		}
	} else {
		max := uint64(1) << bitWidth
		for {
			v := rand.Uint64() % max
			if v != 0 {
				value = v
				break
			}
		}
	}

	return ConstNumber{
		Value:      value,
		BitWidth:   bitWidth,
		Signedness: signed,
	}
}

// RandomConstNumberWithBitWidth generates a random ConstNumber with specific bit width and non-zero value
func RandomConstNumberWithBitWidth(bitWidth int, signed bool) ConstNumber {
	if bitWidth <= 0 || bitWidth > 34 {
		panic("bitWidth must be between 1 and 34")
	}

	var value uint64
	if signed {
		max := int64(1) << (bitWidth - 1)
		for {
			rangeVal := rand.Int63n(2 * max)
			v := int64(rangeVal - max)
			if v != 0 {
				value = uint64(v)
				break
			}
		}
	} else {
		maxv := uint64(1) << bitWidth
		for {
			v := rand.Uint64() % maxv
			if v != 0 {
				value = v
				break
			}
		}
	}

	return ConstNumber{
		Value:      value,
		BitWidth:   bitWidth,
		Signedness: signed,
	}
}
