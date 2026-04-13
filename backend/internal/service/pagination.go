package service

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

func normalizePagination(page, limit *int) (p, l int) {
	p = 1
	if page != nil && *page > 0 {
		p = *page
	}
	l = defaultListLimit
	if limit != nil && *limit > 0 {
		l = *limit
		if l > maxListLimit {
			l = maxListLimit
		}
	}
	return p, l
}
