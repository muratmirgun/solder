package main

type A struct{}
type B struct{}

func NewA(B) A            { return A{} }
func NewB(A) B            { return B{} }
func NewAErr() (A, error) { return A{}, nil }

func main() {}
