package main

import "testing"

func TestRapidTurns(t *testing.T) {
	var g game
	g.reset()
	g.food = cell{0, 0}
	g.turn(cell{0, -1}) // Up.
	g.turn(cell{0, 1})  // Reverse of the queued turn, so ignore it.
	g.turn(cell{-1, 0}) // Left, after moving up.
	for _, want := range []cell{{10, 9}, {9, 9}, {8, 9}} {
		g.step()
		if g.ended || g.snake[0] != want {
			t.Fatalf("head = %v, ended = %t; want head %v and game running", g.snake[0], g.ended, want)
		}
	}
}
