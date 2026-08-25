package model

type TrendPoint struct {
	Seq         int
	Temperature float64
	Pressure    float64
}

type Trend struct {
	ReactorID string
	Points    []TrendPoint
}

func (t *Trend) Add(point TrendPoint) {
	point.Seq = len(t.Points)
	t.Points = append(t.Points, point)
}

func (t *Trend) Size() int {
	return len(t.Points)
}
