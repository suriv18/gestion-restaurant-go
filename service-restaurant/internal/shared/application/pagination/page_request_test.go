package pagination

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct{ in, want PageRequest }{
		{PageRequest{-1, 0}, PageRequest{0, 25}},
		{PageRequest{2, 500}, PageRequest{2, 100}},
		{PageRequest{1, 10}, PageRequest{1, 10}},
	}
	for _, c := range cases {
		if got := c.in.Normalize(); got != c.want {
			t.Errorf("%v -> %v, quería %v", c.in, got, c.want)
		}
	}
	if (PageRequest{3, 10}).Offset() != 30 {
		t.Fatal("offset incorrecto")
	}
}
