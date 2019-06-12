package vec

import (
	"testing"
)

var (
	NULL = Vec3{}
)

func TestBasics(t *testing.T) {
	v := Vec3{1, 2, 3}
	if v[0] != 1 || v[1] != 2 || v[2] != 3 {
		t.Errorf("Vector construction is not obvious")
	}
}

func TestLength(t *testing.T) {
	if NULL.Length() != 0 {
		t.Errorf("Null vector has not 0 length")
	}
	v := Vec3{2, 2, 1}
	if v.Length() != 3 {
		t.Errorf("%v Length is not 3", v)
	}
	v = Vec3{2, 1, 2}
	if v.Length() != 3 {
		t.Errorf("%v Length is not 3", v)
	}
	v = Vec3{1, 2, 2}
	if v.Length() != 3 {
		t.Errorf("%v Length is not 3", v)
	}
}

func TestAdd(t *testing.T) {
	v := Vec3{1, 2, 3}
	got := Add(NULL, v)
	if v != got {
		t.Errorf("Adding a null vector changed the vector")
	}
	got = Add(v, NULL)
	if v != got {
		t.Errorf("Adding a null vector changed the vector")
	}
	got = Add(v, v)
	want := Vec3{2, 4, 6}
	if got != want {
		t.Errorf("Add(%v,%v) = %v want %v", v, v, got, want)
	}
}

func TestSub(t *testing.T) {
	v := Vec3{1, 2, 3}
	got := Sub(v, NULL)
	if v != got {
		t.Errorf("Substracting a null vector changed the vector")
	}
	got = Sub(v, v)
	if got != NULL {
		t.Errorf("Sub(%v,%v) = %v want %v", v, v, got, NULL)
	}
	v2 := Vec3{9, 7, 5}
	got = Sub(v2, v)
	want := Vec3{8, 5, 2}
	if got != want {
		t.Errorf("Sub(%v,%v) = %v want %v", v2, v, got, want)
	}
}

func TestScale(t *testing.T) {
	v := Vec3{1, 2, 3}
	got := Add(NULL, v)
	if v != got {
		t.Errorf("Adding a null vector changed the vector")
	}
	got = Add(v, NULL)
	if v != got {
		t.Errorf("Adding a null vector changed the vector")
	}
	got = Add(v, v)
	want := Vec3{2, 4, 6}
	if got != want {
		t.Errorf("Add(%v,%v) = %v want %v", v, v, got, want)
	}

}

func TestNormalize(t *testing.T) {
}

func TestDot(t *testing.T) {
}

func TestEqual(t *testing.T) {
	v1 := Vec3{2, 3, 4}
	v2 := Vec3{4, 3, 2}
	if v1 != v1 {
		t.Errorf("Vectors are not considered equal to them self")
	}
	if v1 == v2 {
		t.Errorf("Vectors %v and %v are considered equal", v1, v2)
	}
}
