package main

import (
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	adv "github.com/vmaxer/timpark/pkg/adventure"
)

var flags = adv.Flags

func rect(r, g, b, x, y, w, h float64) {
	adv.SetColor(r, g, b)
	adv.FillRect(x, y, w, h)
}

func circle(r, g, b, x, y, radius float64) {
	adv.SetColor(r, g, b)
	adv.FillCircle(x, y, radius)
}

func drawHat(x, y, s float64) {
	rect(0.75, 0.2, 0.17, x-14*s, y-4*s, 28*s, 5*s)
	rect(0.75, 0.2, 0.17, x-8*s, y-20*s, 16*s, 17*s)
	rect(0.15, 0.6, 0.6, x-8*s, y-8*s, 16*s, 4*s)
}

func drawPlayer(x, y, s, facing float64, walking bool) {
	step := 0.0
	if walking {
		step = math.Sin(adv.Time()*12) * 4
	}
	adv.Push()
	adv.Translate(x, y)
	adv.Scale(s*facing, s)
	rect(0.2, 0.22, 0.35, -9+step, -30, 7, 30)
	rect(0.2, 0.22, 0.35, 2-step, -30, 7, 30)
	rect(0.25, 0.55, 0.3, -12, -66, 24, 38)
	rect(0.95, 0.75, 0.6, 9, -60, 6, 20)
	circle(0.95, 0.75, 0.6, 0, -78, 13)
	circle(0.1, 0.1, 0.1, 6, -80, 2)
	rect(0.4, 0.25, 0.1, -13, -92, 22, 8)
	if flags["hatOn"] {
		drawHat(0, -88, 1.1)
	}
	adv.Pop()
}

var crow = &adv.Speaker{X: 320, Y: 130, Color: adv.RGB(0.7, 0.85, 1)}

func drawCrow(x, y float64) {
	circle(0.1, 0.1, 0.12, x, y, 11)
	circle(0.1, 0.1, 0.12, x+10, y-9, 7)
	adv.SetColor(0.9, 0.7, 0.1)
	adv.FillPolygon(x+16, y-10, x+25, y-7, x+16, y-6)
	circle(1, 1, 1, x+12, y-11, 2)
}

func drawKey(x, y float64) {
	adv.SetColor(0.85, 0.7, 0.2)
	adv.SetLineWidth(3)
	adv.LineCircle(x-8, y, 6)
	adv.Line(x-2, y, x+14, y)
	adv.Line(x+10, y, x+10, y+6)
	adv.Line(x+14, y, x+14, y+6)
	adv.SetLineWidth(1)
}

func drawCookie(x, y float64) {
	circle(0.75, 0.5, 0.25, x, y, 10)
	circle(0.3, 0.15, 0.05, x-4, y-3, 2)
	circle(0.3, 0.15, 0.05, x+3, y+2, 2)
	circle(0.3, 0.15, 0.05, x+4, y-4, 1.5)
}

func init() {
	adv.Player.Draw = drawPlayer

	adv.NewItem("cookie", &adv.Item{
		Name: "Cookie",
		Look: "A crunchy cookie. Crows love these.",
		Draw: drawCookie,
	})

	adv.NewItem("key", &adv.Item{
		Name: "Key",
		Look: "A small brass key.",
		Draw: drawKey,
	})

	adv.NewRoom("bedroom", &adv.Room{
		Walk:  []adv.Rect{{X: 20, Y: 245, W: 600, H: 55}},
		Depth: &adv.Depth{Top: 245, Bottom: 300, Far: 0.85, Near: 1},

		Draw: func() {
			rect(0.86, 0.78, 0.62, 0, 0, 640, 240)
			rect(0.55, 0.36, 0.22, 0, 240, 640, 68)
			for x := 0.0; x <= 640; x += 64 {
				rect(0.48, 0.3, 0.18, x, 240, 2, 68)
			}
			rect(0.45, 0.3, 0.2, 0, 232, 640, 8)
			rect(0.4, 0.7, 0.95, 60, 50, 100, 80)
			circle(1, 0.9, 0.3, 130, 75, 14)
			rect(1, 1, 1, 60, 88, 100, 4)
			rect(1, 1, 1, 108, 50, 4, 80)
			adv.SetColor(1, 1, 1)
			adv.SetLineWidth(4)
			adv.LineRect(60, 50, 100, 80)
			rect(0.45, 0.25, 0.6, 270, 60, 60, 80)
			adv.SetColor(0.95, 0.4, 0.6)
			adv.Print("LÖVE", 282, 88)
			rect(0.5, 0.3, 0.15, 30, 200, 180, 50)
			rect(0.95, 0.95, 0.95, 40, 190, 50, 18)
			rect(0.8, 0.25, 0.25, 90, 192, 120, 22)
			rect(0.5, 0.3, 0.15, 220, 195, 40, 50)
			rect(0.4, 0.24, 0.12, 225, 212, 30, 3)
			if !flags["cookieTaken"] {
				drawCookie(240, 186)
			}
			rect(0.45, 0.28, 0.14, 400, 80, 90, 165)
			if flags["wardrobeOpen"] {
				rect(0.15, 0.1, 0.08, 406, 86, 78, 150)
				if !flags["hatTaken"] {
					drawHat(445, 182, 1.3)
				}
			} else {
				rect(0.3, 0.18, 0.1, 444, 86, 2, 150)
				circle(0.9, 0.8, 0.3, 438, 165, 3)
				circle(0.9, 0.8, 0.3, 452, 165, 3)
			}
			rect(0.55, 0.35, 0.2, 560, 100, 60, 140)
			circle(0.9, 0.8, 0.3, 570, 175, 4)
		},

		Hotspots: []*adv.Hotspot{
			{Name: "Window", X: 60, Y: 50, W: 100, H: 80,
				Look: "A sunny day. Perfect weather for wearing a hat.",
				Use:  "I'd rather use the door."},
			{Name: "Poster", X: 270, Y: 60, W: 60, H: 80,
				Look: "A poster of my favourite game engine.",
				Use:  "It's stuck to the wall."},
			{Name: "Bed", X: 30, Y: 190, W: 180, H: 60,
				Look: "My bed. No hat under the pillow, I already checked.",
				Use:  "No time to sleep. My hat is missing!"},
			{Name: "Nightstand", X: 220, Y: 195, W: 40, H: 50,
				Look: "My trusty nightstand.",
				Use:  "Only socks in the drawer."},
			{Name: "Cookie", X: 228, Y: 175, W: 24, H: 22,
				When: func() bool { return !flags["cookieTaken"] },
				Look: "A cookie. Breakfast!",
				Use: func() {
					flags["cookieTaken"] = true
					adv.Take("cookie")
					adv.Say("I'll save it for later.")
				}},
			{Name: "Wardrobe", X: 400, Y: 80, W: 90, H: 165,
				WalkTo: []float64{380, 268},
				Look: func() {
					if flags["wardrobeOpen"] {
						adv.Say("An open wardrobe.")
					} else {
						adv.Say("My wardrobe. I keep it locked, but I lost the key.")
					}
				},
				Use: func() {
					if flags["wardrobeOpen"] {
						adv.Say("It's already open.")
					} else {
						adv.Say("It's locked.")
					}
				},
				Items: map[string]any{
					"key": func() {
						flags["wardrobeOpen"] = true
						adv.Drop("key")
						adv.Say("Click! It's open.")
					},
				}},
			{Name: "Tiny hat", X: 425, Y: 158, W: 40, H: 30,
				WalkTo: []float64{380, 268},
				When:   func() bool { return flags["wardrobeOpen"] && !flags["hatTaken"] },
				Look:   "There it is! My tiny hat!",
				Use: func() {
					flags["hatTaken"] = true
					flags["hatOn"] = true
					adv.Say("Found it! I feel complete again.")
					adv.Pause(0.5)
					adv.TheEnd("The End\n\nThanks for playing!")
				}},
			{Name: "Door to the garden", X: 560, Y: 100, W: 60, H: 140,
				Exit: "garden", Entry: []float64{90, 270}},
		},

		Enter: func() {
			if !flags["started"] {
				flags["started"] = true
				adv.Pause(0.5)
				adv.Say("Good morning!")
				adv.Say("Wait... where is my tiny hat?")
			}
		},
	})

	adv.NewRoom("garden", &adv.Room{
		Walk:  []adv.Rect{{X: 40, Y: 245, W: 580, H: 55}},
		Depth: &adv.Depth{Top: 245, Bottom: 300, Far: 0.85, Near: 1},

		Draw: func() {
			rect(0.55, 0.8, 1, 0, 0, 640, 230)
			circle(1, 0.95, 0.5, 560, 50, 26)
			circle(1, 1, 1, 200, 60, 20)
			circle(1, 1, 1, 225, 55, 26)
			circle(1, 1, 1, 252, 62, 18)
			rect(0.35, 0.65, 0.25, 0, 230, 640, 78)
			rect(0.86, 0.78, 0.62, 0, 40, 110, 200)
			rect(0.55, 0.35, 0.2, 30, 100, 60, 140)
			for x := 150.0; x <= 640; x += 40 {
				rect(0.95, 0.95, 0.9, x, 175, 10, 62)
			}
			rect(0.95, 0.95, 0.9, 150, 188, 490, 7)
			rect(0.95, 0.95, 0.9, 150, 215, 490, 7)
			rect(0.45, 0.3, 0.15, 470, 100, 30, 140)
			circle(0.2, 0.5, 0.2, 485, 80, 50)
			circle(0.25, 0.55, 0.22, 450, 100, 35)
			circle(0.25, 0.55, 0.22, 520, 100, 35)
			rect(0.7, 0.35, 0.2, 200, 230, 30, 25)
			circle(0.9, 0.3, 0.4, 207, 222, 6)
			circle(0.95, 0.85, 0.3, 222, 220, 6)
			if flags["crowFed"] {
				drawCrow(480, 62)
			} else {
				drawCrow(320, 168)
				drawKey(338, 174)
			}
			if flags["crowFed"] && !flags["keyTaken"] {
				drawKey(325, 268)
			}
		},

		Hotspots: []*adv.Hotspot{
			{Name: "Door to my bedroom", X: 30, Y: 100, W: 60, H: 140,
				Exit: "bedroom", Entry: []float64{540, 270}},
			{Name: "Tree", X: 430, Y: 30, W: 110, H: 210,
				Look: "An old apple tree. No apples, and no hats."},
			{Name: "Flowerpot", X: 195, Y: 210, W: 40, H: 45,
				Look: "Geraniums.",
				Use:  "I already looked under it. Only worms."},
			{Name: "Crow", X: 305, Y: 150, W: 40, H: 30,
				When: func() bool { return !flags["crowFed"] },
				Look: "A crow, with a shiny key in its beak. That's my wardrobe key!",
				Use: func() {
					adv.Say("Give me that key!")
					adv.Say("Caw!", crow)
				},
				Items: map[string]any{
					"cookie": func() {
						adv.Drop("cookie")
						flags["crowFed"] = true
						adv.Say("Caw! Caw!", crow)
						adv.Say("It dropped the key and took the cookie up into the tree.")
					},
				}},
			{Name: "Crow", X: 465, Y: 45, W: 40, H: 30,
				When: func() bool { return flags["crowFed"] },
				Look: "A happy crow, munching on my cookie.",
				Use:  func() { adv.Say("Caw.", &adv.Speaker{X: 485, Y: 20, Color: crow.Color}) }},
			{Name: "Key", X: 310, Y: 258, W: 32, H: 20,
				When: func() bool { return flags["crowFed"] && !flags["keyTaken"] },
				Use: func() {
					flags["keyTaken"] = true
					adv.Take("key")
					adv.Say("My wardrobe key!")
				}},
		},
	})
}

type game struct{}

func (g *game) Update() error {
	x, y := ebiten.CursorPosition()
	for button, id := range map[ebiten.MouseButton]int{ebiten.MouseButtonLeft: 1, ebiten.MouseButtonRight: 2} {
		if inpututil.IsMouseButtonJustPressed(button) {
			adv.MousePressed(float64(x), float64(y), id)
		}
		if inpututil.IsMouseButtonJustReleased(button) {
			adv.MouseReleased(float64(x), float64(y), id)
		}
	}
	for _, key := range inpututil.AppendJustPressedKeys(nil) {
		adv.KeyPressed(key)
	}
	return adv.Update(1 / float64(ebiten.TPS()))
}

func (g *game) Draw(screen *ebiten.Image) {
	adv.Draw(screen)
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	adv.SetWindowSize(outsideWidth, outsideHeight)
	return outsideWidth, outsideHeight
}

func main() {
	ebiten.SetWindowTitle("The Tiny Hat")
	ebiten.SetWindowSize(960, 540)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	adv.Init()
	adv.Go("bedroom", 300, 275)
	if err := ebiten.RunGame(&game{}); err != nil {
		log.Fatal(err)
	}
}
