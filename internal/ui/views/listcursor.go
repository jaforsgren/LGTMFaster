package views

const minVisibleListRows = 3

type listCursor struct {
	index       int
	offset      int
	visibleRows int
}

func newListCursor() listCursor {
	return listCursor{visibleRows: minVisibleListRows}
}

func (c *listCursor) SetVisibleRows(rows int) {
	c.visibleRows = max(minVisibleListRows, rows)
	c.scrollToIndex()
}

func (c *listCursor) Index() int {
	return c.index
}

func (c *listCursor) MoveUp() {
	if c.index > 0 {
		c.index--
		c.scrollToIndex()
	}
}

func (c *listCursor) MoveDown(count int) {
	if c.index < count-1 {
		c.index++
		c.scrollToIndex()
	}
}

func (c *listCursor) Reset() {
	c.index = 0
	c.offset = 0
}

func (c *listCursor) SetIndex(index, count int) {
	c.index = index
	c.Clamp(count)
}

func (c *listCursor) Clamp(count int) {
	if c.index >= count {
		c.index = count - 1
	}
	if c.index < 0 {
		c.index = 0
	}
	c.offset = min(c.offset, max(0, count-c.visibleRows))
	c.scrollToIndex()
}

// VisibleRange returns the half-open [start, end) window of item indices to render.
func (c *listCursor) VisibleRange(count int) (int, int) {
	start := min(c.offset, max(0, count-1))
	end := min(start+c.visibleRows, count)
	return start, end
}

func (c *listCursor) scrollToIndex() {
	if c.index < c.offset {
		c.offset = c.index
	}
	if c.index >= c.offset+c.visibleRows {
		c.offset = c.index - c.visibleRows + 1
	}
}
