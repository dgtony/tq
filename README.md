# Task queue
Package implements lightweight task queue with pool of workers in Golang. Allows to enqueue all incoming heavy request and process it with defined number of goroutines.

##Example

Look at the example of simple Fibonacci sequence computation service implementation. Here we use task queue and 2 workers to process incoming calculation requests.


``` go
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dgtony/tq"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
)

// Send incoming channel of the task queue to handlers
func HandlerWrapper(handler func(http.ResponseWriter, *http.Request, chan<- tq.Task), queue chan tq.Task) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(w, r, queue)
	}
}

// Fibonacci calculation job
type FibRequest struct {
	Args   int
	Labor  func(int) ([]int, error)
	Result []int
	Error  error
}

// Implement TaskRequester
func (f *FibRequest) Compute() {
	var err error
	f.Result, err = f.Labor(f.Args)
	if err != nil {
		f.Error = err
	}
}

// JSON-response to the client
type FibResponse struct {
	Arg     int    `json:"arg"`
	Result  []int  `json:"result"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// Fibonacci sequence – heavy computation task in this example!
func FibSeries(n int) ([]int, error) {
	switch {
	case n == 0:
		return []int{0}, nil
	case n == 1:
		return []int{0, 1}, nil
	case n > 92:
		return nil, errors.New(fmt.Sprintf("argument is too big: %d", n))
	default:
		result := []int{0, 1}
		for i := 2; i <= n; i++ {
			result = append(result, result[i-1]+result[i-2])
		}
		return result, nil
	}
}

// API request handler
func FibSeriesHandler(w http.ResponseWriter, r *http.Request, q chan<- tq.Task) {
	n, _ := strconv.Atoi(mux.Vars(r)["num"])
	response := FibResponse{
		Arg: n,
	}
	// Create task
	job := &FibRequest{
		Labor: FibSeries,
		Args:  n,
	}
	task, rCh := tq.CreateTask(job)
	// Send task to workers
	q <- task
	// wait for the result
	tmp := <-rCh
	close(rCh)

	result := tmp.Job.(*FibRequest)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if result.Error != nil {
		response.Error = fmt.Sprint(result.Error)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(response)
		return
	}
	w.WriteHeader(http.StatusOK)
	response.Success = true
	response.Result = result.Result
	_ = json.NewEncoder(w).Encode(response)
}

func main() {
	// create task processing queue with 2 workers
	numWorkers := 2
	queue := tq.TaskQueue(numWorkers)

	// request routing
	router := mux.NewRouter()
	router.HandleFunc("/fib/{num:[0-9]+}", HandlerWrapper(FibSeriesHandler, queue)).Methods("GET")

	log.Println("server started")
	log.Fatal(http.ListenAndServe(":3030", router))
}

```