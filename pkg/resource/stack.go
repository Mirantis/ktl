package resource

type stack[E any] []E

func (s *stack[E]) push(v ...E) {
	for i := len(v) - 1; i >= 0; i-- {
		*s = append(*s, v[i])
	}
}

func (s *stack[E]) pop() E { //nolint:ireturn
	i := len(*s) - 1
	v := (*s)[i]
	*s = (*s)[:i]

	return v
}
