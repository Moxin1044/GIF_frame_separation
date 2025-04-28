package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type tappableImage struct {
	widget.BaseWidget
	image       *canvas.Image
	onPrimary   func()
	onSecondary func()
}

func (t *tappableImage) Tapped(e *fyne.PointEvent) {
	if t.onPrimary != nil {
		t.onPrimary()
	}
}

func (t *tappableImage) TappedSecondary(e *fyne.PointEvent) {
	if t.onSecondary != nil {
		t.onSecondary()
	}
}

func newTappableImage(img image.Image, onPrimary, onSecondary func()) *tappableImage {
	t := &tappableImage{
		image:       canvas.NewImageFromImage(img),
		onPrimary:   onPrimary,
		onSecondary: onSecondary,
	}
	t.image.FillMode = canvas.ImageFillContain
	t.image.SetMinSize(fyne.NewSize(120, 120))
	t.ExtendBaseWidget(t)
	return t
}

func (t *tappableImage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.image)
}

func main() {
	myApp := app.New()
	window := myApp.NewWindow("GIF帧提取器")
	window.Resize(fyne.NewSize(1200, 800))
	// 加载图标资源
	icon := fyne.NewStaticResource("icon", resourceIconPng.StaticContent)
	// 设置窗口或应用图标
	window.SetIcon(icon)
	var gifFrames []image.Image
	currentFrame := canvas.NewImageFromImage(nil)
	currentFrame.FillMode = canvas.ImageFillContain
	currentFrame.SetMinSize(fyne.NewSize(500, 500))
	framesContainer := container.NewGridWithColumns(6)

	openButton := widget.NewButton("打开GIF文件", func() {
		dialog.ShowFileOpen(func(r fyne.URIReadCloser, err error) {
			if err != nil || r == nil {
				return
			}
			defer r.Close()

			gifData, err := gif.DecodeAll(r)
			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			gifFrames = processGifFrames(gifData)

			framesContainer.Objects = nil
			for i, frame := range gifFrames {
				index := i
				thumbnail := container.NewBorder(
					nil,
					widget.NewLabel("帧 "+strconv.Itoa(index+1)),
					nil,
					nil,
					newTappableImage(
						frame,
						func() {
							currentFrame.Image = frame
							currentFrame.Refresh()
						},
						func() {
							saveSingleFrame(frame, index, window)
						},
					),
				)
				framesContainer.Add(thumbnail)
			}

			if len(gifFrames) > 0 {
				currentFrame.Image = gifFrames[0]
				currentFrame.Refresh()
			}
			framesContainer.Refresh()
		}, window)
	})

	saveAllButton := widget.NewButton("保存所有帧", func() {
		if len(gifFrames) == 0 {
			dialog.ShowInformation("提示", "请先打开GIF文件", window)
			return
		}

		dialog.ShowFolderOpen(func(lu fyne.ListableURI, err error) {
			if err != nil || lu == nil {
				return
			}

			for i, frame := range gifFrames {
				path := filepath.Join(lu.Path(), "frame_"+strconv.Itoa(i+1)+".png")
				if err := saveImage(frame, path); err != nil {
					dialog.ShowError(err, window)
					return
				}
			}
			dialog.ShowInformation("完成", "成功保存 "+strconv.Itoa(len(gifFrames))+" 帧到目录\n"+lu.Path(), window)
		}, window)
	})

	controls := container.NewHBox(openButton, saveAllButton, layout.NewSpacer())
	mainContent := container.NewBorder(
		controls,
		nil,
		nil,
		nil,
		container.NewHSplit(
			container.NewScroll(currentFrame),
			container.NewBorder(
				widget.NewLabelWithStyle("缩略图区域（左键预览，右键另存为）", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				nil,
				nil,
				nil,
				container.NewVScroll(framesContainer),
			),
		),
	)

	window.SetContent(mainContent)
	window.ShowAndRun()
}

func processGifFrames(g *gif.GIF) []image.Image {
	var frames []image.Image
	var prevFrame *image.RGBA

	bounds := g.Image[0].Bounds()
	bg := image.NewRGBA(bounds)

	for i, srcImg := range g.Image {
		if i > 0 {
			switch g.Disposal[i-1] {
			case gif.DisposalBackground:
				drawBackground(bg, g)
			case gif.DisposalPrevious:
				if prevFrame != nil {
					bg = prevFrame
				}
			}
		}

		draw.Draw(bg, srcImg.Bounds(), srcImg, srcImg.Bounds().Min, draw.Over)
		frameCopy := image.NewRGBA(bg.Bounds())
		copy(frameCopy.Pix, bg.Pix)

		frames = append(frames, frameCopy)
		prevFrame = bg
	}

	return frames
}

func drawBackground(img *image.RGBA, g *gif.GIF) {
	bgColor := g.Config.ColorModel.(color.Palette)[g.BackgroundIndex]
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)
}

func saveSingleFrame(img image.Image, index int, window fyne.Window) {
	dialog.ShowFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil || w == nil {
			return
		}
		defer w.Close()

		path := w.URI().Path()
		if filepath.Ext(path) == "" {
			path += ".png"
		}

		if err := saveImage(img, path); err != nil {
			dialog.ShowError(err, window)
		} else {
			dialog.ShowInformation("保存成功", "帧已保存至: "+path, window)
		}
	}, window)
}

func saveImage(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}
