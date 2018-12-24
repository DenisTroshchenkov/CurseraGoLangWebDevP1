package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func SingleHash(in, out chan interface{}) {
	var localWg sync.WaitGroup
	var mxMd5 = &sync.Mutex{}
	for val := range in {
		localWg.Add(1)
		go func(data int) {
			defer localWg.Done()
			strVal := strconv.Itoa(data)
			ch1 := make(chan string)
			go func() {
				ch1 <- DataSignerCrc32(strVal)
			}()
			ch2 := make(chan string)
			go func() {
				mxMd5.Lock()
				md5Res := DataSignerMd5(strVal)
				mxMd5.Unlock()
				ch2 <- DataSignerCrc32(md5Res)
			}()
			out <- <-ch1 + "~" + <-ch2
		}(val.(int))
	}
	localWg.Wait()
}

func MultiHash(in, out chan interface{}) {
	const thNum = 6
	var localWg sync.WaitGroup
	for val := range in {
		localWg.Add(1)
		go func(crcData string) {
			defer localWg.Done()
			var mlHashWg sync.WaitGroup
			resultHashes := make([]string, thNum)
			for i := 0; i < thNum; i++ {
				mlHashWg.Add(1)
				go func(th int, data string) {
					defer mlHashWg.Done()
					resultHashes[th] = DataSignerCrc32(strconv.Itoa(th) + data)
				}(i, crcData)
			}
			mlHashWg.Wait()
			out <- strings.Join(resultHashes, "")
		}(val.(string))
	}
	localWg.Wait()
}

func CombineResults(in, out chan interface{}) {
	var resultHashes []string
	for val := range in {
		resultHashes = append(resultHashes, val.(string))
	}
	sort.Strings(resultHashes)
	result := strings.Join(resultHashes, "_")
	out <- result
}

var globWg sync.WaitGroup

func workerWrapper(jobNum int, newJob job, in, out chan interface{}) {
	defer globWg.Done()
	defer close(out)
	fmt.Println("Start new job:", jobNum)
	newJob(in, out)
	fmt.Println("End job:", jobNum)
}

func ExecutePipeline(jobs ...job) {
	var inputChan chan interface{}
	for jobNum, newJob := range jobs {
		outputChan := make(chan interface{}, 20)
		globWg.Add(1)
		go workerWrapper(jobNum, newJob, inputChan, outputChan)
		inputChan = outputChan
	}
	globWg.Wait()
}
