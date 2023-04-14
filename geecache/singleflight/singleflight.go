package singleflight

import "sync"

// 正在进行或已经结束的请求
type call struct {
	wg  sync.WaitGroup //避免重复的请求输入
	val interface{}
	err error
}

// 管理某一key的请求call
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()         //请求进行中，则等待
		return c.val, c.err //请求结束，返回结果
	}
	c := new(call)
	c.wg.Add(1)  //发起请求前加锁
	g.m[key] = c //对应key有请求在处理
	g.mu.Unlock()
	c.val, c.err = fn() //调用fn发起请求
	c.wg.Done()         //请求结束，锁减1
	g.mu.Lock()
	delete(g.m, key) //更新g.m
	g.mu.Unlock()
	return c.val, c.err
}
