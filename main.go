package main

import (
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/mattn/go-tflite"
)

// make user part of video group to access webcam
// sudo usermod -a -G video $USER
//
//	type Game struct {
//		Device *device.Device
//		Frame  *ebiten.Image
//		Stop   context.CancelFunc
//		Width  int
//		Height int
//	}
//
//	func NewGame() (*Game, error) {
//		ctx, cancle := context.WithCancel(context.Background())
//		dev, err := device.Open("/dev/video0", device.WithBufferSize(1), device.WithPixFormat(v4l2.PixFormat{
//			Width:       1280,
//			Height:      720,
//			PixelFormat: v4l2.PixelFmtMJPEG,
//			Field:       v4l2.FieldNone,
//		}))
//		if err != nil {
//			return nil, fmt.Errorf("cannot open device %w", err)
//		}
//		if err := dev.Start(ctx); err != nil {
//			return nil, fmt.Errorf("start device got %w", err)
//		}
//		return &Game{Device: dev, Width: 1280, Height: 720, Stop: cancle}, nil
//	}
//
//	func (g *Game) Close() {
//		if g.Stop != nil {
//			g.Stop()
//		}
//		if g.Device != nil {
//			g.Device.Stop()
//		}
//
// }
//
//	func (g *Game) Update() error {
//		frame := <-g.Device.GetFrames()
//		img, err := jpeg.Decode(bytes.NewReader(frame.Data))
//		if err != nil {
//			frame.Release()
//			return fmt.Errorf("cannot decode %w", err)
//		}
//		g.Frame = ebiten.NewImageFromImage(img)
//		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
//			jpegData := make([]byte, len(frame.Data))
//			copy(jpegData, frame.Data)
//			saveSnapshot(jpegData)
//		}
//		if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
//			return ebiten.Termination
//		}
//		frame.Release()
//		return nil
//	}
//
//	func saveSnapshot(raw []byte) {
//		if err := os.WriteFile("photo.jpg", raw, 0644); err != nil {
//			fmt.Println("save file error ", err)
//			return
//		}
//		fmt.Println("saved")
//	}
//
//	func (g *Game) Draw(screen *ebiten.Image) {
//		if g.Frame != nil {
//			screen.DrawImage(g.Frame, nil)
//		}
//	}
//
//	func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
//		return g.Width, g.Height
//	}
func loadInterpreter(path string) (*tflite.Model, *tflite.Interpreter) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Fatalf("missing model: %s", path)
	}
	model := tflite.NewModelFromFile(path)
	if model == nil {
		log.Fatalf("cannot load model: %s", path)
	}
	opts := tflite.NewInterpreterOptions()
	opts.SetNumThread(4)
	interp := tflite.NewInterpreter(model, opts)
	if interp == nil || interp.AllocateTensors() != tflite.OK {
		log.Fatalf("cannot init interpreter: %s", path)
	}
	return model, interp
}

func main() {
	detModel, detInterp := loadInterpreter("hand_landmarker_extracted/hand_detector.tflite")
	defer detModel.Delete()
	defer detInterp.Delete()

	lmModel, lmInterp := loadInterpreter("hand_landmarker_extracted/hand_landmarks_detector.tflite")
	defer lmModel.Delete()
	defer lmInterp.Delete()

	tracker := NewHandTracker(detInterp, lmInterp)

	ebiten.SetWindowSize(1280, 920)
	game := NewGame(tracker)
	ebiten.SetWindowTitle("Webcam - SPACE to capture -- Q to Close")
	defer game.webCam.Close()
	defer game.mat.Close()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
