package main

import "testing"

func TestFruitScoresAndGrowth(t *testing.T) {
	for _, tc := range []struct {
		roll, points int
	}{
		{0, 1}, {79, 1}, {80, 5}, {94, 5}, {95, 10}, {99, 10},
	} {
		var g game
		g.reset()
		g.food = cell{11, 10}
		g.fruit = fruitForRoll(tc.roll)
		g.turn(cell{1, 0})
		g.step()
		if g.score != tc.points || len(g.snake) != 4 {
			t.Errorf("抽到 %d 時得到 %d 分、長度 %d，預期 %d 分、長度 4", tc.roll, g.score, len(g.snake), tc.points)
		}
	}
}

func TestSpeedByScore(t *testing.T) {
	for _, tc := range []struct{ score, milliseconds int }{
		{0, 115}, {9, 115}, {10, 105}, {13, 105}, {59, 65}, {60, 55}, {100, 55},
	} {
		if got := moveInterval(tc.score); got != tc.milliseconds {
			t.Errorf("%d 分時的移動間隔為 %d 毫秒，預期 %d 毫秒", tc.score, got, tc.milliseconds)
		}
	}
}

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
