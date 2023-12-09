package imaging

import (
	"math"

	"github.com/disintegration/imaging"
)

// Magic Kernel Sharp 2013
// http://johncostella.com/magic/
var mks2013Filter = imaging.ResampleFilter{
	Support: 2.5,
	Kernel: func(x float64) float64 {
		x = math.Abs(x)
		if x >= 2.5 {
			return 0.0
		}
		if x >= 1.5 {
			return -0.125 * (x - 2.5) * (x - 2.5)
		}
		if x >= 0.5 {
			return 0.25 * (4*x*x - 11*x + 7)
		}
		return 1.0625 - 1.75*x*x
	},
}
