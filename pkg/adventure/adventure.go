package adventure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	Width     = 640
	Height    = 360
	BarHeight = 52
)

type Rect struct {
	X, Y, W, H float64
}

type Depth struct {
	Top, Bottom, Far, Near float64
}

type Speaker struct {
	X, Y  float64
	Color Color
}

type Hotspot struct {
	Name       string
	X, Y, W, H float64
	WalkTo     []float64
	When       func() bool
	Look       any
	Use        any
	Items      map[string]any
	Exit       string
	Entry      []float64
}

type Item struct {
	ID    string
	Name  string
	Look  any
	Draw  func(x, y float64)
	Items map[string]any
}

type Room struct {
	ID        string
	Walk      []Rect
	Depth     *Depth
	Draw      func()
	DrawFront func()
	Update    func(dt float64)
	Enter     func()
	Hotspots  []*Hotspot
}

type PlayerState struct {
	X, Y, Speed, Facing float64
	TX, TY              float64
	Moving              bool
	Walking             bool
	Draw                func(x, y, s, facing float64, walking bool)
}

type script struct {
	resume chan bool
	yield  chan func() bool
	done   func() bool
	dead   bool
	err    any
}

type line struct {
	text string
	time float64
	who  *Speaker
}

type noticeLine struct {
	text string
	time float64
}

type abortSignal struct{}

var (
	Rooms     = map[string]*Room{}
	Items     = map[string]*Item{}
	Flags     = map[string]bool{}
	Inventory []string
	Player    = &PlayerState{X: 320, Y: 280, Speed: 140, Facing: 1}
)

var (
	canvas           *ebiten.Image
	chime            *oto.Player
	scale            = 1.0
	offsetX, offsetY float64
	windowW, windowH = 960.0, 540.0
	mouseX, mouseY   float64
	room             *Room
	held             *Item
	label            string
	speech           *line
	notice           *noticeLine
	ending           string
	editing          bool
	dragStart        []float64
	scripts          []*script
	running          *script
	quit             bool
	startTime        = time.Now()
	defaultSpeech    = Color{1, 1, 0.6, 1}
	white            = Color{1, 1, 1, 1}
)

func NewRoom(id string, def *Room) *Room {
	def.ID = id
	if def.Walk == nil {
		def.Walk = []Rect{{0, 0, Width, Height - BarHeight}}
	}
	Rooms[id] = def
	return def
}

func NewItem(id string, def *Item) *Item {
	def.ID = id
	Items[id] = def
	return def
}

func Time() float64 {
	return time.Since(startTime).Seconds()
}

func wait(done func() bool) {
	if running == nil {
		return
	}
	s := running
	s.yield <- done
	if !<-s.resume {
		panic(abortSignal{})
	}
}

func Run(fn func()) {
	s := &script{resume: make(chan bool), yield: make(chan func() bool)}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if _, aborted := r.(abortSignal); aborted {
					return
				}
				s.err = r
			}
			s.dead = true
			s.yield <- nil
		}()
		if !<-s.resume {
			panic(abortSignal{})
		}
		fn()
	}()
	scripts = append(scripts, s)
}

func stepScripts() {
	for i := 0; i < len(scripts); {
		s := scripts[i]
		if s.done == nil || s.done() {
			running = s
			s.resume <- true
			s.done = <-s.yield
			running = nil
			if s.err != nil {
				panic(s.err)
			}
		}
		if s.dead {
			scripts = slices.Delete(scripts, i, i+1)
		} else {
			i++
		}
	}
}

func stopScripts() {
	for _, s := range scripts {
		if !s.dead {
			s.resume <- false
		}
	}
	scripts = nil
}

func Busy() bool {
	return len(scripts) > 0
}

func Say(text string, who ...*Speaker) {
	l := &line{text: text, time: 1.2 + float64(len([]rune(text)))*0.055}
	if len(who) > 0 {
		l.who = who[0]
	}
	speech = l
	wait(func() bool { return speech != l })
}

func Pause(seconds float64) {
	until := Time() + seconds
	wait(func() bool { return Time() >= until })
}

func Notice(text string) {
	notice = &noticeLine{text: text, time: 3}
}

func DepthScale(y float64) float64 {
	if room == nil || room.Depth == nil {
		return 1
	}
	d := room.Depth
	t := math.Max(0, math.Min(1, (y-d.Top)/(d.Bottom-d.Top)))
	return d.Far + (d.Near-d.Far)*t
}

func clampToFloor(x, y float64) (float64, float64) {
	best, bestX, bestY := math.Inf(1), x, y
	for _, r := range room.Walk {
		cx := math.Max(r.X, math.Min(x, r.X+r.W))
		cy := math.Max(r.Y, math.Min(y, r.Y+r.H))
		d := (cx-x)*(cx-x) + (cy-y)*(cy-y)
		if d < best {
			best, bestX, bestY = d, cx, cy
		}
	}
	return bestX, bestY
}

func Walk(x, y float64) {
	p := Player
	p.TX, p.TY = clampToFloor(x, y)
	p.Moving = true
	wait(func() bool { return !p.Moving })
}

func Face(x float64) {
	if x < Player.X {
		Player.Facing = -1
	} else {
		Player.Facing = 1
	}
}

func Go(id string, pos ...float64) {
	r, ok := Rooms[id]
	if !ok {
		panic(fmt.Sprintf("there is no room called %s", id))
	}
	room = r
	p := Player
	x, y := p.X, p.Y
	if len(pos) >= 2 {
		x, y = pos[0], pos[1]
	}
	p.X, p.Y = clampToFloor(x, y)
	p.Moving = false
	held = nil
	if room.Enter != nil {
		Run(room.Enter)
	}
}

func CurrentRoom() *Room {
	return room
}

func Has(id string) bool {
	return slices.Contains(Inventory, id)
}

func Take(id string) {
	if _, ok := Items[id]; !ok {
		panic(fmt.Sprintf("there is no item called %s", id))
	}
	if !Has(id) {
		Inventory = append(Inventory, id)
		if chime != nil {
			chime.Pause()
			chime.Seek(0, io.SeekStart)
			chime.Play()
		}
	}
}

func Drop(id string) {
	if i := slices.Index(Inventory, id); i >= 0 {
		Inventory = slices.Delete(Inventory, i, i+1)
	}
	if held != nil && held.ID == id {
		held = nil
	}
}

func TheEnd(text string) {
	ending = text
}

func respond(reaction any, fallback string) {
	switch r := reaction.(type) {
	case func():
		r()
	case string:
		Say(r)
	default:
		if fallback != "" {
			Say(fallback)
		}
	}
}

func present(spot *Hotspot) bool {
	return spot.When == nil || spot.When()
}

func hotspotAt(x, y float64) *Hotspot {
	for i := len(room.Hotspots) - 1; i >= 0; i-- {
		s := room.Hotspots[i]
		if present(s) && x >= s.X && x < s.X+s.W && y >= s.Y && y < s.Y+s.H {
			return s
		}
	}
	return nil
}

func slotPosition(i int) (float64, float64) {
	return 8 + float64(i)*52, Height - BarHeight + 4
}

func slotAt(x, y float64) int {
	for i := range Inventory {
		sx, sy := slotPosition(i)
		if x >= sx && x < sx+44 && y >= sy && y < sy+44 {
			return i
		}
	}
	return -1
}

func itemAt(x, y float64) *Item {
	if slot := slotAt(x, y); slot >= 0 {
		return Items[Inventory[slot]]
	}
	return nil
}

func interact(spot *Hotspot, item *Item) {
	Run(func() {
		startRoom := room
		wx, wy := spot.X+spot.W/2, spot.Y+spot.H
		if spot.WalkTo != nil {
			wx, wy = spot.WalkTo[0], spot.WalkTo[1]
		}
		Walk(wx, wy)
		if room != startRoom {
			return
		}
		Face(spot.X + spot.W/2)
		if item != nil {
			respond(spot.Items[item.ID], "That doesn't work.")
		} else if spot.Exit != "" {
			Go(spot.Exit, spot.Entry...)
		} else {
			respond(spot.Use, "I can't do anything with that.")
		}
	})
}

func lookAt(spot *Hotspot) {
	Run(func() {
		Face(spot.X + spot.W/2)
		respond(spot.Look, "Nothing special about the "+strings.ToLower(spot.Name)+".")
	})
}

func lookAtItem(item *Item) {
	Run(func() {
		respond(item.Look, "Nothing special about the "+strings.ToLower(item.Name)+".")
	})
}

type saveData struct {
	Room      string          `json:"room"`
	X         float64         `json:"x"`
	Y         float64         `json:"y"`
	Facing    float64         `json:"facing"`
	Flags     map[string]bool `json:"flags"`
	Inventory []string        `json:"inventory"`
}

func savePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "tinyhat-adventure", "save.json")
}

func Save() {
	p := Player
	data, err := json.Marshal(saveData{room.ID, p.X, p.Y, p.Facing, Flags, Inventory})
	if err == nil {
		err = os.MkdirAll(filepath.Dir(savePath()), 0o755)
	}
	if err == nil {
		err = os.WriteFile(savePath(), data, 0o644)
	}
	if err != nil {
		Notice("Could not save")
		return
	}
	Notice("Saved")
}

func Restore() {
	raw, err := os.ReadFile(savePath())
	if err != nil {
		Notice("There is no saved game yet")
		return
	}
	var data saveData
	if err := json.Unmarshal(raw, &data); err != nil {
		Notice("The saved game is damaged")
		return
	}
	stopScripts()
	speech, ending, held = nil, "", nil
	for k := range Flags {
		delete(Flags, k)
	}
	for k, v := range data.Flags {
		Flags[k] = v
	}
	Inventory = slices.Clone(data.Inventory)
	room = Rooms[data.Room]
	p := Player
	p.X, p.Y, p.Facing, p.Moving = data.X, data.Y, data.Facing, false
	Notice("Loaded")
}

func makeChime() (*oto.Player, error) {
	const rate = 22050
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   rate,
		ChannelCount: 1,
		Format:       oto.FormatSignedInt16LE,
	})
	if err != nil {
		return nil, err
	}
	<-ready
	count := int(rate * 0.3)
	buf := make([]byte, count*2)
	for i := 0; i < count; i++ {
		t := float64(i) / rate
		f := 1320.0
		if t < 0.1 {
			f = 880
		}
		v := int16(math.Sin(2*math.Pi*f*t) * 0.25 * (1 - t/0.3) * 32767)
		buf[i*2] = byte(v)
		buf[i*2+1] = byte(uint16(v) >> 8)
	}
	return ctx.NewPlayer(bytes.NewReader(buf)), nil
}

func Init() {
	canvas = ebiten.NewImage(Width, Height)
	loadFonts()
	if source, err := makeChime(); err == nil {
		chime = source
	}
}

func SetWindowSize(w, h int) {
	windowW, windowH = float64(w), float64(h)
}

func updateLabel() {
	label = ""
	if editing {
		label = fmt.Sprintf("%d, %d", int(mouseX), int(mouseY))
		return
	}
	if Busy() || ending != "" {
		return
	}
	var spot *Hotspot
	if mouseY < Height-BarHeight {
		spot = hotspotAt(mouseX, mouseY)
	}
	targetName := ""
	targetIsHeld := false
	if item := itemAt(mouseX, mouseY); item != nil {
		targetName = item.Name
		targetIsHeld = item == held
	} else if spot != nil {
		targetName = spot.Name
	}
	switch {
	case held != nil && targetName != "" && !targetIsHeld:
		label = "Use " + held.Name + " with " + targetName
	case held != nil:
		label = "Use " + held.Name + " with"
	default:
		label = targetName
	}
}

func Update(dt float64) error {
	scale = math.Min(windowW/Width, windowH/Height)
	offsetX, offsetY = (windowW-Width*scale)/2, (windowH-Height*scale)/2
	mx, my := ebiten.CursorPosition()
	mouseX, mouseY = (float64(mx)-offsetX)/scale, (float64(my)-offsetY)/scale

	p := Player
	p.Walking = p.Moving
	if p.Moving {
		dx, dy := p.TX-p.X, p.TY-p.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		step := p.Speed * dt * DepthScale(p.Y)
		if math.Abs(dx) > 1 {
			if dx > 0 {
				p.Facing = 1
			} else {
				p.Facing = -1
			}
		}
		if dist <= step {
			p.X, p.Y, p.Moving = p.TX, p.TY, false
		} else {
			p.X, p.Y = p.X+dx/dist*step, p.Y+dy/dist*step
		}
	}
	if speech != nil {
		speech.time -= dt
		if speech.time <= 0 {
			speech = nil
		}
	}
	if notice != nil {
		notice.time -= dt
		if notice.time <= 0 {
			notice = nil
		}
	}
	if room.Update != nil {
		room.Update(dt)
	}
	stepScripts()
	updateLabel()
	if quit {
		return ebiten.Termination
	}
	return nil
}

func MousePressed(x, y float64, button int) {
	x, y = (x-offsetX)/scale, (y-offsetY)/scale
	if ending != "" {
		return
	}
	if editing {
		if button == 1 {
			dragStart = []float64{math.Floor(x), math.Floor(y)}
		}
		return
	}
	if speech != nil {
		speech = nil
		return
	}
	if Busy() {
		return
	}
	if y >= Height-BarHeight {
		item := itemAt(x, y)
		switch {
		case item == nil:
			held = nil
		case button == 2:
			held = nil
			lookAtItem(item)
		case held != nil && held != item:
			a := held
			held = nil
			Run(func() {
				reaction := a.Items[item.ID]
				if reaction == nil {
					reaction = item.Items[a.ID]
				}
				respond(reaction, "That doesn't work.")
			})
		case held == item:
			held = nil
		default:
			held = item
		}
		return
	}
	spot := hotspotAt(x, y)
	item := held
	held = nil
	if button == 2 {
		if spot != nil {
			lookAt(spot)
		}
	} else if spot != nil {
		interact(spot, item)
	} else {
		p := Player
		p.TX, p.TY = clampToFloor(x, y)
		p.Moving = true
	}
}

func MouseReleased(x, y float64, button int) {
	if !editing || dragStart == nil || button != 1 {
		return
	}
	x, y = math.Floor((x-offsetX)/scale), math.Floor((y-offsetY)/scale)
	x0, y0 := math.Min(dragStart[0], x), math.Min(dragStart[1], y)
	w, h := math.Abs(x-dragStart[0]), math.Abs(y-dragStart[1])
	dragStart = nil
	var text string
	if w > 2 && h > 2 {
		text = fmt.Sprintf("x = %d, y = %d, w = %d, h = %d", int(x0), int(y0), int(w), int(h))
	} else {
		text = fmt.Sprintf("%d, %d", int(x), int(y))
	}
	clipboard.WriteAll(text)
	fmt.Println(text)
	Notice("Copied " + text)
}

func KeyPressed(key ebiten.Key) {
	switch {
	case key == ebiten.KeyEscape:
		quit = true
	case key == ebiten.KeyF2:
		editing = !editing
		dragStart = nil
	case key == ebiten.KeyF5:
		Save()
	case key == ebiten.KeyF9:
		Restore()
	case key == ebiten.KeyF11, key == ebiten.KeyEnter && ebiten.IsKeyPressed(ebiten.KeyAlt):
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
}

func outlined(text string, x, y, width float64, align Align, c Color) {
	SetColor(0, 0, 0)
	for dx := -1.0; dx <= 1; dx++ {
		for dy := -1.0; dy <= 1; dy++ {
			if dx != 0 || dy != 0 {
				Printf(text, x+dx, y+dy, width, align)
			}
		}
	}
	setColor(c)
	Printf(text, x, y, width, align)
}

func drawSpeech() {
	p := Player
	x := p.X
	top := p.Y - 100*DepthScale(p.Y)
	c := defaultSpeech
	if who := speech.who; who != nil {
		x, top, c = who.X, who.Y, who.Color
	}
	width := 280.0
	lines := wrapText(font, speech.text, width)
	y := math.Max(4, top-float64(len(lines))*fontHeight(font))
	x = math.Max(4, math.Min(Width-width-4, x-width/2))
	outlined(speech.text, x, y, width, AlignCenter, c)
}

func drawBar() {
	top := float64(Height - BarHeight)
	SetColor(0.12, 0.1, 0.14)
	FillRect(0, top, Width, BarHeight)
	for i, id := range Inventory {
		x, y := slotPosition(i)
		item := Items[id]
		if item == held {
			SetColor(0.45, 0.4, 0.25)
		} else {
			SetColor(0.22, 0.2, 0.25)
		}
		FillRoundedRect(x, y, 44, 44, 4)
		if item != held {
			SetColor(1, 1, 1)
			item.Draw(x+22, y+22)
		}
	}
}

func drawEditor() {
	SetLineWidth(1)
	for _, r := range room.Walk {
		SetColor(0.2, 1, 0.3, 0.8)
		LineRect(r.X+0.5, r.Y+0.5, r.W, r.H)
	}
	for _, s := range room.Hotspots {
		alpha := 0.35
		if present(s) {
			alpha = 0.9
		}
		SetColor(1, 0.9, 0.2, alpha)
		LineRect(s.X+0.5, s.Y+0.5, s.W, s.H)
		Print(s.Name, s.X+2, s.Y+1)
	}
	if dragStart != nil {
		SetColor(0.3, 0.9, 1)
		LineRect(dragStart[0]+0.5, dragStart[1]+0.5, mouseX-dragStart[0], mouseY-dragStart[1])
	}
	outlined("Editor: drag to copy a rectangle, click to copy a point, F2 to play", 0, Height-BarHeight-20, Width, AlignCenter, Color{0.6, 1, 1, 1})
}

func Draw(screen *ebiten.Image) {
	target = canvas
	canvas.Clear()
	canvas.Fill(Color{0, 0, 0, 1}.rgba())
	SetColor(1, 1, 1)
	room.Draw()
	p := Player
	SetColor(1, 1, 1)
	p.Draw(p.X, p.Y, DepthScale(p.Y), p.Facing, p.Walking)
	if room.DrawFront != nil {
		SetColor(1, 1, 1)
		room.DrawFront()
	}
	drawBar()
	if editing {
		drawEditor()
	}
	if speech != nil {
		drawSpeech()
	}
	if label != "" {
		outlined(label, 0, 6, Width, AlignCenter, white)
	}
	if notice != nil {
		outlined(notice.text, 0, Height-BarHeight-40, Width-8, AlignRight, Color{0.6, 1, 0.6, 1})
	}
	if held != nil {
		SetColor(1, 1, 1)
		held.Draw(mouseX+14, mouseY+14)
	}
	if ending != "" {
		SetColor(0, 0, 0, 0.75)
		FillRect(0, 0, Width, Height)
		activeFont = bigFont
		lines := wrapText(bigFont, ending, Width-40)
		outlined(ending, 20, (Height-float64(len(lines))*fontHeight(bigFont))/2, Width-40, AlignCenter, Color{1, 0.85, 0.4, 1})
		activeFont = font
	}
	screen.Fill(Color{0, 0, 0, 1}.rgba())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(offsetX, offsetY)
	screen.DrawImage(canvas, op)
}
