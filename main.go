package main

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
	"image/jpeg"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/vladimirvivien/go4vl/device"
	"github.com/vladimirvivien/go4vl/v4l2"
	"golang.org/x/image/font/gofont/goregular"
)

// make user part of video group to access webcam
// sudo usermod -a -G video $USER
type Game struct {
	Device *device.Device
	Frame  *ebiten.Image
	Stop   context.CancelFunc
	Width  int
	Height int

	countdown     int
	countdownTick int
	countdownFace *text.GoTextFace
}

func NewGame() (*Game,error) {
	ctx, cancle :=context.WithCancel(context.Background())
	dev, err := device.Open("/dev/video0", device.WithBufferSize(1),device.WithPixFormat(v4l2.PixFormat{
		Width:       1280,
		Height:      720,
		PixelFormat: v4l2.PixelFmtMJPEG,
		Field:       v4l2.FieldNone,
}))
	if err != nil {
		return nil, fmt.Errorf("cannot open device %w",err)
	}
	if err := dev.Start(ctx); err != nil {
		return nil,fmt.Errorf("start device got %w", err)
	}
	faceSrc, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		dev.Close()
		cancle()
		return nil, fmt.Errorf("load font: %w", err)
	}

	return &Game{
		Device: dev,
		Width:  1280,
		Height: 720,
		Stop:   cancle,
		countdownFace: &text.GoTextFace{
			Source: faceSrc,
			Size:   120,
		},
	}, nil
}
func (g *Game)Close(){
if g.Stop!=nil{
	g.Stop()}
if g.Device!=nil{
     g.Device.Stop()
}

}
func (g *Game) Update() error {
	frame := <-g.Device.GetFrames()
	img, err := jpeg.Decode(bytes.NewReader(frame.Data))
	if err != nil {
		frame.Release()
		return fmt.Errorf("cannot decode %w", err)
	}
	g.Frame = ebiten.NewImageFromImage(img)

	if g.countdown > 0 {
		g.countdownTick++
		if g.countdownTick >= ebiten.TPS() {
			g.countdownTick = 0
			if g.countdown == 1 {
				jpegData := make([]byte, len(frame.Data))
				copy(jpegData, frame.Data)
				saveSnapshot(jpegData)
				g.countdown = 0
			} else {
				g.countdown--
			}
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.countdown = 3
		g.countdownTick = 0
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		return ebiten.Termination
	}
	frame.Release()
	return nil
}
func saveSnapshot(raw []byte) {
	if err := os.WriteFile("photo.jpg", raw, 0644); err != nil {
		fmt.Println("save file error ", err)
		return
	}
	fmt.Println("saved")
}
func (g *Game) Draw(screen *ebiten.Image) {
	if g.Frame != nil {
		screen.DrawImage(g.Frame, nil)
	}
	if g.countdown > 0 {
		vector.DrawFilledRect(screen, 0, 0, float32(g.Width), float32(g.Height), color.RGBA{0, 0, 0, 80}, false)
		msg := fmt.Sprintf("%d", g.countdown)
		w, h := text.Measure(msg, g.countdownFace, 0)
		x := float64(g.Width)/2 - w/2
		y := float64(g.Height)/2 - h/2
		op := &text.DrawOptions{}
		op.GeoM.Translate(x, y)
		op.ColorScale.ScaleWithColor(color.RGBA{255, 255, 255, 255})
		text.Draw(screen, msg, g.countdownFace, op)
	}
}
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.Width, g.Height
}
func main() {
	game,err := NewGame()
	if err!=nil{
     log.Fatalf("new game gotta -> %w",err)
	}
	defer game.Close()
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("Webcam - SPACE: 3s countdown capture | Q: quit")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

