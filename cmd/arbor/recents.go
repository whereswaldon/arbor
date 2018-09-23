package main

type RecentList struct {
	recents []string
	index   int
	full    bool
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
		full:    false,
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
			if !r.full && r.index == len(r.recents) {
    				r.full = true
			}
			r.index %= len(r.recents)
		case <-r.reqData:
    			buflen := r.index
    			if r.full {
        			buflen = len(r.recents)
    			}
    			res := make([]string, buflen)
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
