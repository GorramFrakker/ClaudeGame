package game

import "math"

// Vec2 is a simple 2D vector used for positions and velocities.
type Vec2 struct{ X, Y float64 }

// V constructs a vector.
func V(x, y float64) Vec2 { return Vec2{x, y} }

func (a Vec2) Add(b Vec2) Vec2    { return Vec2{a.X + b.X, a.Y + b.Y} }
func (a Vec2) Sub(b Vec2) Vec2    { return Vec2{a.X - b.X, a.Y - b.Y} }
func (a Vec2) Mul(s float64) Vec2 { return Vec2{a.X * s, a.Y * s} }
func (a Vec2) Len() float64       { return math.Hypot(a.X, a.Y) }
func (a Vec2) Len2() float64      { return a.X*a.X + a.Y*a.Y }

// Norm returns a unit vector in the same direction (zero stays zero).
func (a Vec2) Norm() Vec2 {
	l := a.Len()
	if l == 0 {
		return Vec2{}
	}
	return Vec2{a.X / l, a.Y / l}
}

// Rot rotates the vector by rad radians (clockwise in screen space, y-down).
func (a Vec2) Rot(rad float64) Vec2 {
	s, c := math.Sincos(rad)
	return Vec2{a.X*c - a.Y*s, a.X*s + a.Y*c}
}

// Angle returns the heading of the vector in radians.
func (a Vec2) Angle() float64 { return math.Atan2(a.Y, a.X) }

// FromAngle builds a vector of magnitude mag pointing along rad.
func FromAngle(rad, mag float64) Vec2 {
	s, c := math.Sincos(rad)
	return Vec2{c * mag, s * mag}
}

func dist(a, b Vec2) float64  { return a.Sub(b).Len() }
func dist2(a, b Vec2) float64 { return a.Sub(b).Len2() }

// circlesOverlap reports whether two circles intersect.
func circlesOverlap(a Vec2, ra float64, b Vec2, rb float64) bool {
	r := ra + rb
	return dist2(a, b) <= r*r
}

func clampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampi(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// approach moves cur toward target by at most step, without overshooting.
func approach(cur, target, step float64) float64 {
	if cur < target {
		cur += step
		if cur > target {
			return target
		}
		return cur
	}
	if cur > target {
		cur -= step
		if cur < target {
			return target
		}
		return cur
	}
	return cur
}

// angleTo returns the angle from a to b.
func angleTo(a, b Vec2) float64 { return b.Sub(a).Angle() }

func absf(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxi(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func mini(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// wrapPi wraps an angle to [-pi, pi].
func wrapPi(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}
