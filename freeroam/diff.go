package freeroam

type slotInfo struct {
	JustAdded bool
	Client    *Client
}

type ArrayDiffResult struct {
	Kept    []*Client
	Added   []*Client
	Removed []*Client
}

func ArrayDiff(old []*Client, new []*Client) ArrayDiffResult {
	res := ArrayDiffResult{
		Kept:    make([]*Client, 0),
		Added:   make([]*Client, 0),
		Removed: make([]*Client, 0),
	}
	for _, av := range new {
		if has(old, av) {
			res.Kept = append(res.Kept, av)
		} else {
			res.Added = append(res.Added, av)
		}
	}
	for _, av := range old {
		if !has(new, av) {
			res.Removed = append(res.Removed, av)
		}
	}
	return res
}

func has(a []*Client, v *Client) bool {
	for _, av := range a {
		if av == v {
			return true
		}
	}
	return false
}
