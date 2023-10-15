package slices

import "github.com/zeebo/errs"

func Split[T any](n int, v []T) ([][]T, error) {
	if len(v) == 0 {
		return [][]T{}, nil
	}

	if n <= 0 {
		return nil, errs.New("n:%d must be greater than zero", n)
	}

	div := make(map[int][]T, n)
	for i, val := range v {
		idx := i % n
		l := div[idx]
		l = append(l, val)
		div[idx] = l
	}

	var b [][]T
	for _, v := range div {
		b = append(b, v)
	}

	return b, nil
}
