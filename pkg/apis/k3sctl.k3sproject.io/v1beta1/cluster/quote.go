package cluster

type quoter interface {
	ShellQuote(s string) string
}

func quote(q quoter, s string) string {
	return q.ShellQuote(s)
}
