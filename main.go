package main

import "math/rand/v2"

const (
	columns = 24
	rows    = 20
)

type cell struct{ x, y int }

type game struct {
	snake                       []cell
	direction, food             cell
	pendingTurns                [2]cell
	turnCount                   int
	score, best                 int
	started, paused, ended, won bool
}

func (g *game) reset() {
	*g = game{
		snake:     []cell{{10, 10}, {9, 10}, {8, 10}},
		direction: cell{1, 0},
		best:      g.best,
	}
	g.placeFood()
}

func (g *game) placeFood() {
	var free []cell
	for y := range rows {
		for x := range columns {
			p := cell{x, y}
			if !contains(g.snake, p) {
				free = append(free, p)
			}
		}
	}
	if len(free) == 0 {
		g.ended, g.won = true, true
		return
	}
	g.food = free[rand.IntN(len(free))]
}

func contains(body []cell, p cell) bool {
	for _, segment := range body {
		if segment == p {
			return true
		}
	}
	return false
}

func (g *game) turn(d cell) {
	if g.paused || g.ended || g.turnCount == len(g.pendingTurns) {
		return
	}
	last := g.direction
	if g.turnCount > 0 {
		last = g.pendingTurns[g.turnCount-1]
	}
	if d == (cell{-last.x, -last.y}) {
		return
	}
	g.started = true
	if d != last {
		g.pendingTurns[g.turnCount] = d
		g.turnCount++
	}
}

func (g *game) step() {
	if !g.started || g.paused || g.ended {
		return
	}
	if g.turnCount > 0 {
		g.direction = g.pendingTurns[0]
		copy(g.pendingTurns[:], g.pendingTurns[1:g.turnCount])
		g.turnCount--
	}
	head := cell{g.snake[0].x + g.direction.x, g.snake[0].y + g.direction.y}
	eating := head == g.food
	body := g.snake
	if !eating {
		body = body[:len(body)-1]
	}
	if head.x < 0 || head.x >= columns || head.y < 0 || head.y >= rows || contains(body, head) {
		g.ended = true
		return
	}
	g.snake = append(body, cell{})
	copy(g.snake[1:], body)
	g.snake[0] = head
	if eating {
		g.score++
		g.placeFood()
	}
}

func (g *game) status() string {
	switch {
	case g.won:
		return "你填滿了整個棋盤！按 Enter 再玩一次"
	case g.ended:
		return "遊戲結束！按 Enter 再玩一次"
	case g.paused:
		return "暫停中，按空白鍵繼續"
	case !g.started:
		return "按方向鍵或 WASD 開始"
	default:
		return "方向鍵 / WASD 移動　空白鍵暫停　R 重開"
	}
}
