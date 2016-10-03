//performing memory recycling using functions
package main

import (
	"container/list"
	"fmt"
	"math/rand"
	"runtime"
	"time"
)

var makes int
var frees int

func makeBuffer() []byte {
	makes++
	return make([]byte, rand.Intn(5000000)+5000000)
}

// a buffer with timestamp
type queued struct {
	when  time.Time
	slice []byte
}

func makeRecycler() (get, give chan []byte) {
	get = make(chan []byte)
	give = make(chan []byte)

	go func() {
		q := new(list.List)
		for {
			// list is empty
			if q.Len() == 0 {
				q.PushFront(queued{when: time.Now(), slice: makeBuffer()})
			}

			// get the front of the queue
			e := q.Front()
			timeout := time.NewTimer(time.Minute)
			select {
			// receive a buffer and push it to the list
			case b := <-give:
				timeout.Stop()
				q.PushFront(queued{when: time.Now(), slice: b})
				// send a buffer
			case get <- e.Value.(queued).slice:
				timeout.Stop()
				q.Remove(e)
				// remove a buffer if it's too old. this will unblock every minute
			case <-timeout.C:
				e := q.Front()
				for e != nil {
					n := e.Next()
					if time.Since(e.Value.(queued).when) > time.Minute {
						q.Remove(e)
						e.Value = nil
					}
					e = n
				}
			}
		}

	}()

	return
}

func main() {
	pool := make([][]byte, 20)

	get, give := makeRecycler()

	var m runtime.MemStats
	for {
		// get a buffer from the list
		b := <-get
		i := rand.Intn(len(pool))
		if pool[i] != nil {
			// and give it back for further retrieval
			give <- pool[i]
		}

		pool[i] = b
		time.Sleep(time.Second)

		bytes := 0
		for i := 0; i < len(pool); i++ {
			if pool[i] != nil {
				bytes += len(pool[i])
			}
		}

		runtime.ReadMemStats(&m)
		fmt.Printf("%d,%d,%d,%d,%d,%d,%d\n", m.HeapSys, bytes, m.HeapAlloc,
			m.HeapIdle, m.HeapReleased, makes, frees)
	}
}
