package internal

import "math"

type Vector2D struct {
	X float64
	Y float64
}

func (self Vector2D) Sub(vec Vector2D) Vector2D {
	return Vector2D{
		X: self.X - vec.X,
		Y: self.Y - vec.Y,
	}
}

func (self Vector2D) Abs() Vector2D {
	return Vector2D{
		X: math.Abs(self.X),
		Y: math.Abs(self.Y),
	}
}

func (self Vector2D) Length() float64 {
	return math.Sqrt(math.Pow(self.X, 2) + math.Pow(self.Y, 2))
}
