package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

// make user part of video group to access webcam
// sudo usermod -a -G video $USER
//type Game struct {
//	Device *device.Device
//	Frame  *ebiten.Image
//	Stop   context.CancelFunc
//	Width  int
//	Height int
//}
//
//func NewGame() (*Game, error) {
//	ctx, cancle := context.WithCancel(context.Background())
//	dev, err := device.Open("/dev/video0", device.WithBufferSize(1), device.WithPixFormat(v4l2.PixFormat{
//		Width:       1280,
//		Height:      720,
//		PixelFormat: v4l2.PixelFmtMJPEG,
//		Field:       v4l2.FieldNone,
//	}))
//	if err != nil {
//		return nil, fmt.Errorf("cannot open device %w", err)
//	}
//	if err := dev.Start(ctx); err != nil {
//		return nil, fmt.Errorf("start device got %w", err)
//	}
//	return &Game{Device: dev, Width: 1280, Height: 720, Stop: cancle}, nil
//}
//func (g *Game) Close() {
//	if g.Stop != nil {
//		g.Stop()
//	}
//	if g.Device != nil {
//		g.Device.Stop()
//	}
//
//}
//func (g *Game) Update() error {
//	frame := <-g.Device.GetFrames()
//	img, err := jpeg.Decode(bytes.NewReader(frame.Data))
//	if err != nil {
//		frame.Release()
//		return fmt.Errorf("cannot decode %w", err)
//	}
//	g.Frame = ebiten.NewImageFromImage(img)
//	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
//		jpegData := make([]byte, len(frame.Data))
//		copy(jpegData, frame.Data)
//		saveSnapshot(jpegData)
//	}
//	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
//		return ebiten.Termination
//	}
//	frame.Release()
//	return nil
//}
//func saveSnapshot(raw []byte) {
//	if err := os.WriteFile("photo.jpg", raw, 0644); err != nil {
//		fmt.Println("save file error ", err)
//		return
//	}
//	fmt.Println("saved")
//}
//func (g *Game) Draw(screen *ebiten.Image) {
//	if g.Frame != nil {
//		screen.DrawImage(g.Frame, nil)
//	}
//}
//func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
//	return g.Width, g.Height
//}
func main() {
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("Puzzle-Cam Go - Step 1: Webcam")
	game:= NewGame()
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowTitle("Webcam - SPACE to capture -- Q to Close")
	defer game.webCam.Close()
	defer game.mat.Close()
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
