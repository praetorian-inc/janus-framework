package chain

type Strictness int

const (
	Moderate Strictness = iota // Kill chain on Process errors
	Lax                        // Do not kill chain on any error
	Strict                     // Kill chain on Process or Conversion errors
)

func (s Strictness) String() string {
	return []string{"Moderate", "Lax", "Strict"}[s]
}
