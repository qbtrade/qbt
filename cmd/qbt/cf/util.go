package cf

type baseType interface {
	~int | ~uint | ~int8 | ~uint8 | ~int16 | ~uint16 | ~int32 | ~uint32 | ~int64 | ~uint64 | ~float32 | ~float64
}

func Max[T baseType](a T, others ...T) T {
	if len(others) == 0 {
		return a
	}

	maxVal := a
	for i := range others {
		if maxVal < others[i] {
			maxVal = others[i]
		}
	}

	return maxVal
}

func Min[T baseType](a T, others ...T) T {
	if len(others) == 0 {
		return a
	}

	minVal := a
	for i := range others {
		if minVal > others[i] {
			minVal = others[i]
		}
	}

	return minVal
}

func Sum[T baseType](list []T) T {
	if len(list) == 0 {
		return 0
	}
	sum := T(0)
	for _, v := range list {
		sum += v
	}
	return sum
}

func Mean[T baseType](list []T) T {
	if len(list) == 0 {
		return 0
	}
	sum := Sum(list)
	res := sum / T(len(list))
	return res
}
