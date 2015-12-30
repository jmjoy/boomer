package boomer

import (
	"fmt"
	"sort"
	"time"
)

type ResponseTime struct {
	Second float64
	Count  int
	BarLen int
}

type ReportResult struct {
	Summary struct {
		TotalSecond    float64
		SlowestSecond  float64
		FastestSecond  float64
		AverageSecond  float64
		RequestsPerSec float64
		TotalSize      int64
		SizePerRequest int64
	}

	StatusCodeDist map[int]int
	ResponseTimes  []ResponseTime
	LatencyDist    map[string]float64
	ErrorDist      map[string]int
}

type report struct {
	avgTotal float64
	fastest  float64
	slowest  float64
	average  float64
	rps      float64

	results chan *result
	total   time.Duration

	errorDist      map[string]int
	statusCodeDist map[int]int
	lats           []float64
	sizeTotal      int64
}

func newReport(size int, results chan *result, total time.Duration) *report {
	return &report{
		results:        results,
		total:          total,
		statusCodeDist: make(map[int]int),
		errorDist:      make(map[string]int),
	}
}

func (r *report) finalize() *ReportResult {
	for {
		select {
		case res := <-r.results:
			if res.err != nil {
				r.errorDist[res.err.Error()]++
			} else {
				r.lats = append(r.lats, res.duration.Seconds())
				r.avgTotal += res.duration.Seconds()
				r.statusCodeDist[res.statusCode]++
				if res.contentLength > 0 {
					r.sizeTotal += res.contentLength
				}
			}
		default:
			r.rps = float64(len(r.lats)) / r.total.Seconds()
			r.average = r.avgTotal / float64(len(r.lats))
			return r.print()
		}
	}
}

func (r *report) print() *ReportResult {
	result := new(ReportResult)

	sort.Float64s(r.lats)

	if len(r.lats) > 0 {
		r.fastest = r.lats[0]
		r.slowest = r.lats[len(r.lats)-1]

		// Summary
		result.Summary.TotalSecond = r.total.Seconds()
		result.Summary.SlowestSecond = r.slowest
		result.Summary.FastestSecond = r.fastest
		result.Summary.AverageSecond = r.average
		result.Summary.RequestsPerSec = r.rps

		if r.sizeTotal > 0 {
			result.Summary.TotalSize = r.sizeTotal
			result.Summary.SizePerRequest = r.sizeTotal / int64(len(r.lats))
		}

		r.printStatusCodes(result)
		r.printHistogram(result)
		r.printLatencies(result)

	}

	if len(r.errorDist) > 0 {
		r.printErrors(result)
	}

	return result
}

// Prints percentile latencies.
func (r *report) printLatencies(result *ReportResult) {
	pctls := []int{10, 25, 50, 75, 90, 95, 99}
	data := make([]float64, len(pctls))
	j := 0
	for i := 0; i < len(r.lats) && j < len(pctls); i++ {
		current := i * 100 / len(r.lats)
		if current >= pctls[j] {
			data[j] = r.lats[i]
			j++
		}
	}
	result.LatencyDist = make(map[string]float64)
	for i := 0; i < len(pctls); i++ {
		if data[i] > 0 {
			result.LatencyDist[fmt.Sprintf("%v%%", pctls[i])] = data[i]
		}
	}
}

func (r *report) printHistogram(result *ReportResult) {
	bc := 10
	buckets := make([]float64, bc+1)
	counts := make([]int, bc+1)
	bs := (r.slowest - r.fastest) / float64(bc)
	for i := 0; i < bc; i++ {
		buckets[i] = r.fastest + bs*float64(i)
	}
	buckets[bc] = r.slowest
	var bi int
	var max int
	for i := 0; i < len(r.lats); {
		if r.lats[i] <= buckets[bi] {
			i++
			counts[bi]++
			if max < counts[bi] {
				max = counts[bi]
			}
		} else if bi < len(buckets)-1 {
			bi++
		}
	}

	result.ResponseTimes = make([]ResponseTime, 0, 6)
	for i := 0; i < len(buckets); i++ {
		// Normalize bar lengths.
		var barLen int
		if max > 0 {
			barLen = counts[i] * 40 / max
		}

		result.ResponseTimes = append(result.ResponseTimes, ResponseTime{
			Second: buckets[i],
			Count:  counts[i],
			BarLen: barLen,
		})
	}
}

// Prints status code distribution.
func (r *report) printStatusCodes(result *ReportResult) {
	result.StatusCodeDist = r.statusCodeDist
}

func (r *report) printErrors(result *ReportResult) {
	result.ErrorDist = r.errorDist
}
