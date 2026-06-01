package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// drawPoly renders a closed polygon defined in local space at pos, with an
// optional neon outline and a soft glow behind it. local points are scaled,
// rotated, then translated.
func drawPoly(dst *ebiten.Image, local []Vec2, pos Vec2, rot, scale float64, fill, outline color.RGBA, glowR float64, glowCol color.RGBA, glowI float64) {
	if glowR > 0 {
		drawGlow(dst, pos.X, pos.Y, glowR, glowCol, glowI)
	}
	var path vector.Path
	for i, lp := range local {
		wp := lp.Mul(scale).Rot(rot).Add(pos)
		if i == 0 {
			path.MoveTo(float32(wp.X), float32(wp.Y))
		} else {
			path.LineTo(float32(wp.X), float32(wp.Y))
		}
	}
	path.Close()

	if fill.A > 0 {
		var fop vector.FillOptions
		var d1 vector.DrawPathOptions
		d1.AntiAlias = true
		d1.ColorScale.ScaleWithColor(fill)
		vector.FillPath(dst, &path, &fop, &d1)
	}
	if outline.A > 0 {
		var sop vector.StrokeOptions
		sop.Width = 1.8
		sop.LineJoin = vector.LineJoinRound
		sop.LineCap = vector.LineCapRound
		var d2 vector.DrawPathOptions
		d2.AntiAlias = true
		d2.ColorScale.ScaleWithColor(outline)
		vector.StrokePath(dst, &path, &sop, &d2)
	}
}

// pcircle draws a filled anti-aliased circle (small convenience wrapper).
func pcircle(dst *ebiten.Image, pos Vec2, r float64, c color.RGBA) {
	vector.DrawFilledCircle(dst, float32(pos.X), float32(pos.Y), float32(r), c, true)
}

// regularPoly returns the vertices of a regular n-gon of radius r, rotated by
// rot0, in local space.
func regularPoly(n int, r, rot0 float64) []Vec2 {
	pts := make([]Vec2, 0, n)
	for i := 0; i < n; i++ {
		a := rot0 + float64(i)/float64(n)*2*math.Pi
		pts = append(pts, FromAngle(a, r))
	}
	return pts
}

// starPoly returns a star with n points alternating between outer radius r and
// inner radius ri.
func starPoly(n int, r, ri, rot0 float64) []Vec2 {
	pts := make([]Vec2, 0, n*2)
	for i := 0; i < n*2; i++ {
		rr := r
		if i%2 == 1 {
			rr = ri
		}
		a := rot0 + float64(i)/float64(n*2)*2*math.Pi
		pts = append(pts, FromAngle(a, rr))
	}
	return pts
}
