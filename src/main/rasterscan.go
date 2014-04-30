package main

import (
	"image"
	. "image/color"
	"image/png"
	"math"
	"math/cmplx"
	"os"
	"fmt"
	"github.com/nfnt/resize"
)

const SIZE int = 1600

type Range struct {
	low, high float64
}

type Coordinate struct {
	r float64
	i float64
	size float64
	
	rangeR, rangeI Range
}

func NewCoordinate(r, i, size float64) *Coordinate {
	coord := new(Coordinate)
	coord.r = r
	coord.i = i
	coord.size = size
	
	halfSize := size/2

	coord.rangeR = Range{r-halfSize, r+halfSize}
	coord.rangeI = Range{i-halfSize, i+halfSize}

	return coord;
}

func convert(x, y int, location *Coordinate) (float64, float64) {
	r := float64(x) * location.size / float64(SIZE) + location.rangeR.low
	i := float64(y) * location.size / float64(SIZE) + location.rangeI.low
	
	return r, i
}

func Hsv(H, S, V float64) Color {
	Hp := H/60.0
	C := V*S
	X := C*(1.0-math.Abs(math.Mod(Hp, 2.0)-1.0))
	
	m := V-C;
	r, g, b := 0.0, 0.0, 0.0
	
	switch {
	case 0.0 <= Hp && Hp < 1.0: r = C; g = X
	case 1.0 <= Hp && Hp < 2.0: r = X; g = C
	case 2.0 <= Hp && Hp < 3.0: g = C; b = X
	case 3.0 <= Hp && Hp < 4.0: g = X; b = C
	case 4.0 <= Hp && Hp < 5.0: r = X; b = C
	case 5.0 <= Hp && Hp < 6.0: r = C; b = X
	}
	
	f := func(value float64) uint16 {return uint16(value*65535.0)}

	return RGBA64{f(m+r), f(m+g), f(m+b), 0xFFFF}
}

func draw(img *image.RGBA, x, y int, color Color) {
	img.Set(x, SIZE-1-y, color)
} 

func plot(img *image.RGBA, location *Coordinate, maxIterations int) {
	for x := 0; x < SIZE; x++ {
		for y := 0; y < SIZE; y++ {
			r, i := convert(x, y, location)
			c := complex(r, i)
			
			escapeRadius := 10.0
			iterations := 1
			
			z := c

			// Used for period detection
			iterationsFast := 1
			z_fast := z
			
			dz := complex(1,0)
			
			// Distance Estimator Algorithm
			for cmplx.Abs(z) <= escapeRadius && iterations < maxIterations {
				z2 := z*z + c
				dz = 2*z*dz + 1
				z = z2
				iterations = iterations + 1

				// Period detection
				z_fast = z_fast * z_fast + c
				z_fast = z_fast * z_fast + c
				iterationsFast = iterationsFast + 1

				if z == z_fast {
					//period := iterationsFast - iterations
					iterations = maxIterations
				}
			}
			
			magZ := cmplx.Abs(z)
			distanceEstimate := magZ * math.Log(magZ) / cmplx.Abs(dz)
			D := float64(iterations)
			
			if (iterations < maxIterations) {
				D += math.Log2(math.Log2(magZ)) - math.Log2(math.Log2(escapeRadius))
			}

			dwell := math.Floor(D)
			finalrad := D - math.Floor(D)
			finalang := cmplx.Phase(z)
			
			// The Distance Estimate is scaled based on the size of the image and
			// the magnification factor
			dscale := math.Log2(distanceEstimate / location.size * (float64(SIZE)/2))

			if int(D) >= maxIterations {
				draw(img, x, y, White)
			} else {
				// Coloring is achieved by combining DEM, Dwell and Binary Decomposition
				//
				// The following algorithm is an implementation of the pseudocode found
				// on Robert Munafo's website: http://mrob.com/pub/muency/color.html
				var value float64
				
				if dscale > 0 {
					value = 1.0
				} else if dscale > -8 {
					value = (8 + dscale) / 8
				} else {
					value = 0
				}

				P := math.Log(D)/math.Log(100000)

				var angle, radius float64

				if P<0.5 {
					P = 1.0 - 1.5*P
					angle = 1-P
					radius = math.Sqrt(P)
				} else {
					P = 1.5*P - 0.5
					angle = P
					radius = math.Sqrt(P)
				}

				if int(dwell) % 2 != 0 {
					value = 0.85 * value
					radius = 0.667 * radius
				}

				if finalang < 0 {
					angle = angle + 0.02
				}

				angle = angle + 0.0001 * finalrad
				hue := angle * 5.0
				hue = hue - math.Floor(hue)
				// Hue is now in the range [0..1] and needs to be
				// converted to [0..360[
				hue *= 359

				saturation := radius - math.Floor(radius)

				draw(img, x, y, Hsv(hue, saturation, value))
			}
		}
	}
}

func main() {
	
	min := image.Point{0,0}
	max := image.Point{SIZE, SIZE}

	rect := image.Rectangle{min, max}

	img := image.NewRGBA(rect)
	
	//plot(img, NewCoordinate(-0.9189, 0.2646, 0.0625), 10000)
	//plot(img, NewCoordinate(-1.74876455,0,2.4e-5), 10000)
	//plot(img, NewCoordinate(-1.768620774, 0.002428273, 9e-6), 10000)
	//plot(img, NewCoordinate(-0.75, 0, 3.5), 1000)
	//plot(img, NewCoordinate(-1.76733, 0.00002, 0.05), 10000)
	//plot(img, NewCoordinate(-0.7454291059302, 0.1130087933933, 1.8e-10), 10000)
	//plot(img, NewCoordinate(-1.7499254344, 0.0000000003, 2.975e-7), 10000)
	//plot(img, NewCoordinate(-0.744539860355905, 0.121723773894425, 7.5e-12), 10000)
	//plot(img, NewCoordinate(-0.7469449763063739, 0.10787392545265853, 5e-6), 10000)
	//plot(img, NewCoordinate(+0.2925755, -0.0149977, +0.0005), 10000)
	//plot(img, NewCoordinate(-0.74453985651, +0.12172377365, 3.072e-8), 50000)
	//plot(img, NewCoordinate(-.743643887037151, .131825904205330, .000000000051299), 10000)
	//plot(img, NewCoordinate(-0.5434, 0.6132, 0.19), 10000)
	//plot(img, NewCoordinate(-1.2565061941546132, 0.3961765209994203, 2e-4), 10000)
	plot(img, NewCoordinate(-1.9427191355269593, -3.51354564072135E-4, 1e-12), 10000)

	downscaled := resize.Resize(uint(SIZE/2), 0, img, resize.Lanczos3)

	file, err := os.Create("out.png")
	
	if err != nil {
		fmt.Println(err)
		return
	}

	err = png.Encode(file, downscaled)
	if err != nil {
		fmt.Println(err)
	}

}