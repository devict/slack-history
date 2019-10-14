package main

// Pair is used as a helper for sorting a map[string]int by value
type Pair struct {
	Key   string
	Value int
}

// PairList is used as a helper for sorting a map[string]int by value
type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Less(i, j int) bool { return p[i].Value < p[j].Value }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
