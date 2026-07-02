package main

import (
	"fmt"

	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"gocv.io/x/gocv"
)

type Game struct {
	webCam *gocv.VideoCapture
	frame  *ebiten.Image
	mat    *gocv.Mat
}

func NewGame() *Game {
	//0 -> default webcam
	webCam, err := gocv.OpenVideoCapture(0)
	if err != nil {
		return nil
	}
	x := gocv.NewMat()
	return &Game{
		webCam: webCam,
		mat:    &x,
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
	if g.frame == nil {
		g.frame = ebiten.NewImageFromImage(img)
	} else {
		g.frame = ebiten.NewImageFromImage(img)
	}
	return nil
}
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 1280, 720 // Fixed logical resolution
}
func (g *Game) Draw(screen *ebiten.Image) {
	if g.frame == nil {
		screen.Fill(color.RGBA{0, 0, 0, 255})
		fmt.Println("debug......")
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(0.8, 0.8)
	screen.DrawImage(g.frame, op)
}
