package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"gocv.io/x/gocv"
)

type Game struct {
	webCam             *gocv.VideoCapture
	frame              *ebiten.Image
	mat                *gocv.Mat
	liveImage          *ebiten.Image
	capturedImage      *ebiten.Image // after image -> pin top left
	piceces            []*PuzzleImg
	gameState          string // live, puzzle, done
	gridCols, gridRows int
	dragIndex          int
	offsetX, offsetY   float64
	handTracker        *HandTracker
	handCount          int
}

func NewGame(handTracker *HandTracker) *Game {
	// 0 -> default webcam
	webCam, err := gocv.OpenVideoCapture(0)
	if err != nil {
		return nil
	}
	x := gocv.NewMat()
	return &Game{
		webCam:      webCam,
		mat:         &x,
		gridCols:    3,
		gridRows:    3,
		gameState:   "live",
		dragIndex:   -1,
		handTracker: handTracker,
	}
}

func (g *Game) mouseHandlerDrag() {
	mX, mY := ebiten.CursorPosition()
	// Start drag
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		for i := len(g.piceces) - 1; i >= 0; i-- {
			p := g.piceces[i]
			if mX >= int(p.X) && mX < int(p.X)+p.Width &&
				mY >= int(p.Y) && mY < int(p.Y)+p.Height {
				g.dragIndex = i
				g.offsetX = float64(mX) - p.X
				g.offsetY = float64(mY) - p.Y
				return
			}
		}
	}

	if g.dragIndex >= 0 && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		p := g.piceces[g.dragIndex]
		p.X = float64(mX) - g.offsetX
		p.Y = float64(mY) - g.offsetY
	}

	if g.dragIndex >= 0 && inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		p := g.piceces[g.dragIndex]
		dx := p.X - p.TargetX
		dy := p.Y - p.TargetY
		if dx*dx+dy*dy < 8000 {
			p.X = p.TargetX
			p.Y = p.TargetY
		}
		g.dragIndex = -1
	}
}

func (g *Game) Update() error {
	if g.gameState == "live" {
		g.webCam.Read(g.mat)
		if !g.mat.Empty() {
			if g.handTracker != nil {
				g.handCount = g.handTracker.Process(g.mat)
			}
			img, _ := g.mat.ToImage()
			g.liveImage = ebiten.NewImageFromImage(img)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) && g.liveImage != nil {
			g.capturePuzzle()
			g.gameState = "puzzle"
		}
	} else if g.gameState == "puzzle" {
		g.mouseHandlerDrag()
		g.checkStatus()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		return ebiten.Termination
	}

	return nil
}

func (g *Game) capturePuzzle() {
	g.capturedImage = g.liveImage
	g.piceces = nil
	pieceW := g.capturedImage.Bounds().Dx() / g.gridCols
	pieceH := g.capturedImage.Bounds().Dy() / g.gridRows
	for i := 0; i < g.gridCols*g.gridRows; i++ {
		col := i % g.gridCols
		row := i / g.gridCols
		subRect := image.Rect(col*pieceW, row*pieceH, (col+1)*pieceW, (row+1)*pieceH)
		subImg := g.capturedImage.SubImage(subRect).(*ebiten.Image)
		piece := &PuzzleImg{
			Image:   subImg,
			Width:   pieceW,
			Height:  pieceH,
			Index:   i,
			TargetX: float64(col * pieceW),
			TargetY: float64(row * pieceH),
			X:       float64(col*pieceW + rand.Intn(450) - 100), // shuffle a bit
			Y:       float64(row*pieceH + rand.Intn(280) - 100 + 380),
		}
		g.piceces = append(g.piceces, piece)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 1280, 920 // Fixed logical resolution
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{25, 25, 35, 255})

	if g.gameState == "live" && g.liveImage != nil {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(0.5, 0.5)
		screen.DrawImage(g.liveImage, op)
		ebitenutil.DebugPrint(screen, fmt.Sprintf("hands: %d | Space: capture | Q: quit", g.handCount))
	} else if g.capturedImage != nil {
		capOp := &ebiten.DrawImageOptions{}

		capOp.GeoM.Scale(0.5, 0.5)

		screen.DrawImage(g.capturedImage, capOp)
		for i, p := range g.piceces {
			pop := &ebiten.DrawImageOptions{}
			pop.GeoM.Scale(0.5, 0.5)
			pop.GeoM.Translate(p.X, p.Y)
			if i == g.dragIndex {
				pop.ColorScale.Scale(1.25, 1.25, 1.25, 1)
			}
			screen.DrawImage(p.Image, pop)
		}
		if g.gameState == "done" {
			ebitenutil.DebugPrintAt(screen, "COMPLETO! 🎉", 480, 120)
		} else {
			ebitenutil.DebugPrintAt(screen, "Moving with mouse: Q to Quit", 320, 120)
		}
	}
}

func sin(x float64) float64 {
	return float64(math.Sin(x))
}

func (g *Game) checkStatus() {
	done := true
	for _, p := range g.piceces {
		if p.X != p.TargetX || p.Y != p.TargetY {
			done = false
			break
		}
	}
	if done {
		g.gameState = "done"
	}
}
