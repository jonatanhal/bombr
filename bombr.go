package main

import (
	"strconv"
	"io"
	"image"
	"os"
	"time"
	"math"
	"encoding/csv"

	_ "image/png"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"

	"golang.org/x/image/colornames"
)




/** notes

super bomberman2 stage is 13 blocks high, 15 blocks wide

bombr player sprites: 0-29
bg tiles: 30+

*/

type (
	animState int
	FieldCell struct {
		T int
		V pixel.Vec
		R pixel.Rect
	}

	bombrPhys struct {
		runSpeed	float64
		rect		pixel.Rect
		vel		pixel.Vec
	}

	bombrAnim struct {
		sheet	pixel.Picture
		anims	map[string][]pixel.Rect
		rate	float64

		state	animState
		counter	float64
		dir	float64

		frame	pixel.Rect

		sprite	*pixel.Sprite
	}
)

const (
	ScreenWidth  = 960
	ScreenHeight = 896

	idle = iota
	running = iota
	
	bgBlock = 0
	bgCrate = 1
	bgEmpty = -1
)

func run() {
	var (
		cfg	pixelgl.WindowConfig
		win	*pixelgl.Window
		sheet	pixel.Picture
		anims	map[string][]pixel.Rect
		err	error
		field	[]FieldCell
		// todo: last
	)

	field = make([]FieldCell, 195)

	for x := 0; x < 15; x++ {
		for y := 0; y < 13; y++ {
			var t int = bgBlock
			if y == 0 || y == 12 || x == 0 || x == 14 {
				// border blocks
				t = bgBlock
			} else if x % 2 == 0 && y % 2 == 0  {
				// inner blocks
				t = bgBlock
			} else {
				t = bgCrate
			}
			c := FieldCell{
				T: t,
				V: pixel.Vec{
					X: (64.0*float64(x))+32,
					Y: (64.0*float64(y))+32,
				},
				R: pixel.Rect{
					Max: pixel.Vec{X: 64.0*float64(x),Y: 64.0*float64(y),},
					Min: pixel.Vec{X: (64.0*float64(x)) + 32.0,Y: (64.0*float64(y)) + 32.0,},
				},
			}
			field = append(field, c)
		}
	}

	cfg = pixelgl.WindowConfig{
		Title: "bombr",
		Bounds: pixel.R(0,0,ScreenWidth,ScreenHeight),
		VSync: true,
	}

	phys := &bombrPhys{
		runSpeed: 80,
		rect: pixel.R(-64,-64,64,64),
	}

	sheet, anims, err = loadAnimationSheet("bombr-sprite-0001.png", "sheet.csv", 32)
	if err != nil {
		panic(err)
	}

	anim := &bombrAnim{
		sheet: sheet,
		anims: anims,
		rate: 1.0/10,
		dir: +1,
	}

	win, err = pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}
	

	last := time.Now()
	for !win.Closed() {
		var (
			dt float64
			batch *pixel.Batch
		)

		// was: dt
		dt = time.Since(last).Seconds()
		last = time.Now()

		ctrl := pixel.ZV
		if win.Pressed(pixelgl.KeyLeft) {
			ctrl.X--
		}
		if win.Pressed(pixelgl.KeyRight) {
			ctrl.X++
		}
		if win.Pressed(pixelgl.KeyUp) {
			ctrl.Y++
		}
		if win.Pressed(pixelgl.KeyDown) {
			ctrl.Y--
		}

		phys.update(dt, ctrl)
		anim.update(dt, phys)

		win.Clear(colornames.Yellowgreen)
		batch = pixel.NewBatch(&pixel.TrianglesData{}, sheet)
		batch.Clear()

		bgSetup:
		for _, cell := range field {
			if cell.T == bgEmpty {
				continue bgSetup
			}
			bgMat := pixel.IM
			bgMat = bgMat.Scaled(pixel.ZV, 2)
			bgMat = bgMat.Moved(cell.V)
			sprite := pixel.NewSprite(sheet, anims["BG"][cell.T])
			sprite.Draw(batch, bgMat)
		}
		batch.Draw(win)
		anim.draw(win,phys)

		win.Update()
	}
}

func (ba *bombrAnim) update(dt float64, phys *bombrPhys) {
	ba.counter += dt
	
	// determine the new animation state
	var newState animState
	switch {
	case phys.vel.Len() == 0:
		newState = idle
	case phys.vel.Len() > 0:
		newState = running
	}

	// reset the time counter if state changed
	if ba.state != newState {
		ba.state = newState
		ba.counter = 0
	}

	// determine the correct animation frame
	switch ba.state {
	case idle:
		i := int(math.Floor(ba.counter / ba.rate))
		ba.frame = ba.anims["Idle"][i%len(ba.anims["Idle"])]
	case running:
		i := int(math.Floor(ba.counter / ba.rate))
		ba.frame = ba.anims["Run"][i%len(ba.anims["Run"])]
	}

	// set the facing direction of the gopher
	if phys.vel.X != 0 {
		if phys.vel.X > 0 {
			ba.dir = +1
		} else {
			ba.dir = -1
		}
	}
}

func (ba bombrAnim) draw(t pixel.Target, phys *bombrPhys) {
	if ba.sprite == nil {
		ba.sprite = pixel.NewSprite(nil, pixel.Rect{})
	}

	// draw the correct frame with the correct position and direction
	ba.sprite.Set(ba.sheet, ba.frame)
	ba.sprite.Draw(t, pixel.IM.ScaledXY(pixel.ZV, pixel.V(
		phys.rect.W()/ba.sprite.Frame().W(),
		phys.rect.H()/ba.sprite.Frame().H(),
	)).ScaledXY(pixel.ZV, pixel.V(-ba.dir, 1)).Moved(phys.rect.Center()))
}

func (bp *bombrPhys) update(dt float64, ctrl pixel.Vec) {
	switch {
	case ctrl.X < 0:
		bp.vel.X = -bp.runSpeed
	case ctrl.X > 0:
		bp.vel.X = +bp.runSpeed
	case ctrl.Y < 0:
		bp.vel.Y = -bp.runSpeed
	case ctrl.Y > 0:
		bp.vel.Y = +bp.runSpeed
	default:
		bp.vel.X = 0
		bp.vel.Y = 0
	}

	// apply velocity
	bp.rect = bp.rect.Moved(bp.vel.Scaled(dt))

	// todo: check collisions
}

func loadAnimationSheet(path, descPath string, frameWidth float64) (sheet pixel.Picture, anims map[string][]pixel.Rect, err error) {
	var (
		sheetFile *os.File
		descFile *os.File
		sheetImg image.Image
		desc *csv.Reader
	)

	sheetFile, err = os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer sheetFile.Close()
	
	sheetImg, _, err = image.Decode(sheetFile)
	if err != nil {
		return nil, nil, err
	}

	sheet = pixel.PictureDataFromImage(sheetImg)

	var frames []pixel.Rect
	for x := 0.0; x+frameWidth <= sheet.Bounds().Max.X; x += frameWidth {
		frames = append(frames, pixel.R(
			x,
			0,
			x+frameWidth,
			sheet.Bounds().H(),
		))
	}

	descFile, err = os.Open(descPath)
	if err != nil {
		return nil, nil, err
	}
	defer descFile.Close()

	anims = make(map[string][]pixel.Rect)
	desc = csv.NewReader(descFile)
	for {
		anim, err := desc.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}

		name := anim[0]
		start, _ := strconv.Atoi(anim[1])
		end, _ := strconv.Atoi(anim[2])

		anims[name] = frames[start:end+1]
	}
	
	return sheet, anims, nil
}

func main() {
	pixelgl.Run(run)
}
