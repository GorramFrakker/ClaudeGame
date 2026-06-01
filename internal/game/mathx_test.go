package game

import (
	"math"
	"testing"
)

func almost(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestVecBasics(t *testing.T) {
	a := V(3, 4)
	if a.Len() != 5 {
		t.Fatalf("Len = %v, want 5", a.Len())
	}
	if a.Len2() != 25 {
		t.Fatalf("Len2 = %v, want 25", a.Len2())
	}
	n := a.Norm()
	if !almost(n.Len(), 1) {
		t.Fatalf("Norm length = %v, want 1", n.Len())
	}
	z := Vec2{}.Norm()
	if z != (Vec2{}) {
		t.Fatalf("Norm of zero = %v, want zero", z)
	}
}

func TestFromAngleAndRot(t *testing.T) {
	v := FromAngle(0, 2)
	if !almost(v.X, 2) || !almost(v.Y, 0) {
		t.Fatalf("FromAngle(0,2) = %v", v)
	}
	r := V(1, 0).Rot(math.Pi / 2)
	if !almost(r.X, 0) || !almost(r.Y, 1) {
		t.Fatalf("Rot 90 = %v, want (0,1)", r)
	}
}

func TestClampAndApproach(t *testing.T) {
	if clampf(5, 0, 3) != 3 || clampf(-1, 0, 3) != 0 || clampf(2, 0, 3) != 2 {
		t.Fatal("clampf wrong")
	}
	if approach(0, 10, 3) != 3 {
		t.Fatal("approach up wrong")
	}
	if approach(0, 2, 5) != 2 {
		t.Fatal("approach overshoot up wrong")
	}
	if approach(10, 0, 3) != 7 {
		t.Fatal("approach down wrong")
	}
	if approach(1, 5, 0) != 1 {
		t.Fatal("approach zero step wrong")
	}
}

func TestCirclesOverlap(t *testing.T) {
	if !circlesOverlap(V(0, 0), 5, V(8, 0), 5) {
		t.Fatal("should overlap")
	}
	if circlesOverlap(V(0, 0), 1, V(10, 0), 1) {
		t.Fatal("should not overlap")
	}
}

func TestWrapPi(t *testing.T) {
	if !almost(wrapPi(3*math.Pi), math.Pi) && !almost(wrapPi(3*math.Pi), -math.Pi) {
		t.Fatalf("wrapPi(3pi) = %v", wrapPi(3*math.Pi))
	}
	if w := wrapPi(0); w != 0 {
		t.Fatalf("wrapPi(0) = %v", w)
	}
}
