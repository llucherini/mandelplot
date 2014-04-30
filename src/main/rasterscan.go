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
	"strconv"
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

func distanceEstimator(c complex128, maxIterations int, escapeRadius float64) (float64, float64, float64) {
	
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
	dwell := float64(iterations)
	
	if (iterations < maxIterations) {
		dwell += math.Log2(math.Log2(magZ)) - math.Log2(math.Log2(escapeRadius))
	}

	return distanceEstimate, dwell, cmplx.Phase(z)
	
}

func plot(img *image.RGBA, location *Coordinate, maxIterations int) {
	for x := 0; x < SIZE; x++ {
		for y := 0; y < SIZE; y++ {
			r, i := convert(x, y, location)
			c := complex(r, i)
			
			distanceEstimate, D, phase := distanceEstimator(c, maxIterations, 10.0)
			
			dwell := math.Floor(D)
			finalrad := D - dwell
	
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
				
				// Convert the scaled distance estimate to a brightness (called
				// "value" in the HSV color space) from 0.0 to 1.0 in 8 intervals.
				if dscale > 0 {
					value = 1.0
				} else if dscale > -8 {
					value = (8 + dscale) / 8
				} else {
					value = 0
				}
				
				/*The following re-maps the range 0.0...1.0 onto an "angle"
				 and "radius" on the color wheel. This color wheel has white in the
				 center, pastel hues close to the center, and vivid colors around the
				 edge of the wheel. In order to gain maximum use of all the available
				 colors, we compute an angle and radius in a way that places the
				 resulting points equally far apart from one another. The use
				 of square root means that the radius increases more slowly as we
				 move away from the center. Note also that the angle will be multiplied
				 by a constant below, so we'll end up going around the wheel multiple times*/
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
				
				// Brighness and saturation are lowered a bit
				// in every other "stripe".
				if int(dwell) % 2 != 0 {
					value = 0.85 * value
					radius = 0.667 * radius
				}
				
				// This breaks the stripe into "squares" to make
				// external angles evident.
				if phase < 0 {
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
	
	usage := fmt.Sprintf("Usage: %v <real> <img> <size>", os.Args[0]) 

	getFloat := func(arg string) float64 {
		f, err := strconv.ParseFloat(arg, 64)
		if (err != nil) {
			fmt.Println(err)
			fmt.Println(usage)
			os.Exit(1)
		}

		return f
	}

	var r,i,size float64
	
	if len(os.Args) < 4 {
		// Discard arguments and plot the set centered and zoomed out
		r = -0.75
		i = 0
		size = 3.5
	} else {
		r = getFloat(os.Args[1])
		i = getFloat(os.Args[2])
		size = getFloat(os.Args[3])
	}
	
	plot(img, NewCoordinate(r,i,size), 10000)
	
	// The final image is half the size of the generated one.
	// This way we get a crisp anti-aliased image
	downscaled := resize.Resize(uint(SIZE/2), 0, img, resize.Lanczos3)

	file, err := os.Create("out.png")
	
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
		
	err = png.Encode(file, downscaled)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}