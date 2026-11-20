package main

import (
	"fmt"
	"strings"

	"example.com/app/greet"
)

type Prefix string
type Message struct{ Text string }

func Shout(g greet.Greeter, p Prefix) Message { return Message{string(p) + g.Greet()} }
func Loud(m Message) Message                  { return Message{strings.ToUpper(m.Text)} }

type Inner struct{ M Message }
type Pair struct{ Plain, Decorated Message }

func NewInner(m Message) Inner         { return Inner{m} }
func NewPair(m Message, in Inner) Pair { return Pair{m, in.M} }

func main() {
	p := NewPairApp("bob")
	fmt.Println(p.Plain.Text)
	fmt.Println(p.Decorated.Text)
}
