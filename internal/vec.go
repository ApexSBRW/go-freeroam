package internal

import "math"

type Vector2D struct {
	X int64
	Y int64
}

func (self Vector2D) Sub(vec Vector2D) Vector2D {
	return Vector2D{
		X: self.X - vec.X,
		Y: self.Y - vec.Y,
	}
}

func (self Vector2D) Abs() Vector2D {
	return Vector2D{
		X: absI64(self.X),
		Y: absI64(self.Y),
	}
}

func (self Vector2D) Length() int64 {
	tsqrt := rootI64(self.X) + rootI64(self.Y)
	return int64(math.Sqrt(float64(tsqrt)))
}

func absI64(i int64) int64 {
	if i < 0 {
		return -i
	}
	return i
}

func rootI64(i int64) int64 {
	return i * i
}
