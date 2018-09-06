package main

type RecentList struct {
	recents []string
	index   int
	add     chan string
	reqData chan struct{}
	data    chan []string
}

func NewRecents(size int) *RecentList {
	r := &RecentList{
		recents: make([]string, size),
		add:     make(chan string),
		reqData: make(chan struct{}),
		data:    make(chan []string),
		index:   0,
	}
	go r.dispatch()
	return r
}

func (r *RecentList) dispatch() {
	for {
		select {
		case id := <-r.add:
			r.recents[r.index] = id
			r.index++
			r.index %= len(r.recents)
		case <-r.reqData:
			res := make([]string, len(r.recents))
			copy(res, r.recents)
			r.data <- res
		}
	}
}

func (r *RecentList) Add(id string) {
	r.add <- id
}

func (r *RecentList) Data() []string {
	r.reqData <- struct{}{}
	return <-r.data
}
