package main

import "github.com/hajimehoshi/ebiten/v2"

type PuzzleImg struct {
	X, Y                 float64
	Image                *ebiten.Image
	TargetX              float64
	TargetY              float64
	Width, Height, Index int
}
