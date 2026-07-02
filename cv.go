package main

import (
	"fmt"
	"image"
	"image/color"
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
	puzzleImage        *ebiten.Image // ref image captured
	piceces            []*PuzzleImg
	gameState          string // live, puzzle
	gridCols, gridRows int
}

func NewGame() *Game {
	//0 -> default webcam
	webCam, err := gocv.OpenVideoCapture(0)
	if err != nil {
		return nil
	}
	x := gocv.NewMat()
	return &Game{
		webCam:    webCam,
		mat:       &x,
		gridCols:  3,
		gridRows:  3,
		gameState: "live",
	}
}

func (g *Game) Update() error {

	g.webCam.Read(g.mat)
	if g.mat.Empty() {
		return fmt.Errorf("init mat ->%s")
	}
	img, err := g.mat.ToImage()
	if err != nil {
		return fmt.Errorf("can not get image from g.mat() %s", err)
	}
	g.frame = ebiten.NewImageFromImage(img)
	//take picture
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) && g.gameState == "live" {
		g.capturePuzzle()
		g.gameState = "puzzle"
	}
	return nil
}
func (g *Game) capturePuzzle() {
	g.puzzleImage = g.frame
	g.piceces = nil
	pieceW := g.puzzleImage.Bounds().Dx() / g.gridCols
	pieceH := g.puzzleImage.Bounds().Dy() / g.gridRows
	for i := 0; i < g.gridCols*g.gridRows; i++ {
		col := i % g.gridCols
		row := i / g.gridCols
		subRect := image.Rect(col*pieceW, row*pieceH, (col+1)*pieceW, (row+1)*pieceH)
		subImg := g.puzzleImage.SubImage(subRect).(*ebiten.Image)
		piece := &PuzzleImg{
			Image:   subImg,
			Width:   pieceW,
			Height:  pieceH,
			Index:   i,
			TargetX: float64(col * pieceW),
			TargetY: float64(row * pieceH),
			X:       float64(col*pieceW + rand.Intn(200) - 100), // shuffle a bit
			Y:       float64(row*pieceH + rand.Intn(200) - 100),
		}
		g.piceces = append(g.piceces, piece)
	}

}
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 1280, 720 // Fixed logical resolution
}
func (g *Game) Draw(screen *ebiten.Image) {
	if g.frame == nil {
		screen.Fill(color.Black)
		fmt.Println("debug......")
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(0.6, 0.6)
	screen.DrawImage(g.frame, op)
	if g.gameState == "live" {
		ebitenutil.DebugPrint(screen, "Press Space to take picture")
	} else if g.puzzleImage != nil {
		for _, p := range g.piceces {
			pop := &ebiten.DrawImageOptions{}
			pop.GeoM.Translate(p.X, p.Y+250)
			screen.DrawImage(p.Image, pop)
		}
		ebitenutil.DebugPrintAt(screen, "Puzzle Mode", 10, 10)
	}
}
