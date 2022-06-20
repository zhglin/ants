// MIT License

// Copyright (c) 2018 Andy Pan

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package ants

import (
	"runtime"
	"time"
)

// goWorker is the actual executor who runs the tasks,
// it starts a goroutine that accepts tasks and
// performs function calls.
// 是运行任务的实际执行程序，它启动接受任务并执行函数调用的go例程。
type goWorker struct {
	// pool who owns this worker.
	// 上层pool
	pool *Pool

	// task is a job should be done.
	// 可能是有缓冲区，缓冲区长度=1
	task chan func()

	// recycleTime will be updated when putting a worker back into queue.
	// 开始空闲的时间
	recycleTime time.Time
}

// run starts a goroutine to repeat the process
// that performs the function calls.
// 启动一个goroutine来重复执行函数调用的过程。
func (w *goWorker) run() {
	w.pool.addRunning(1)
	go func() {
		// 协程 return时的函数
		defer func() {
			w.pool.addRunning(-1)     // 减少计数
			w.pool.workerCache.Put(w) // 放入缓冲区
			if p := recover(); p != nil {
				if ph := w.pool.options.PanicHandler; ph != nil { // 自定义函数处理
					ph(p)
				} else { // 默认记录日志
					w.pool.options.Logger.Printf("worker exits from a panic: %v\n", p)
					var buf [4096]byte
					n := runtime.Stack(buf[:], false)
					w.pool.options.Logger.Printf("worker exits from panic: %s\n", string(buf[:n]))
				}
			}
			// Call Signal() here in case there are goroutines waiting for available workers.
			// 在这里调用Signal()，以防有等待可用工作者的goroutine。
			w.pool.cond.Signal()
		}()

		// 执行task
		for f := range w.task {
			if f == nil { // reset()函数写入nil
				return
			}

			f()

			// 如果返回失败，就直接return，防止内存泄漏
			if ok := w.pool.revertWorker(w); !ok {
				return
			}
		}
	}()
}
