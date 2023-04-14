package lru

import "container/list"

// 缓存淘汰采用最近最少使用LRU算法，并发访问不安全
type Cache struct {
	maxBytes  int64 //允许使用的最大内存
	nbytes    int64 //当前已使用的内存
	ll        *list.List
	cache     map[string]*list.Element      //字典
	OnEvicted func(key string, value Value) //某条记录被移除时的回调函数
}

type entry struct { //键值对
	key   string
	value Value
}

type Value interface {
	Len() int //返回值所占用的内存字节大小
}

// New为缓存的构造函数
func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// 实现查找功能
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele) //将链表中的对应节点 ele 移动到队尾，这里约定双向链表的front为队尾
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// 实现删除最近最少访问的节点（队首）
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back() //取双向链表队首节点
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)                                //从字典中删除该节点的映射关系
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len()) //更新使用内存
		if c.OnEvicted != nil {                                //调用回调函数
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// 实现节点添加/更新
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok { //键已存在
		c.ll.MoveToFront(ele)    //将该节点移至队尾
		kv := ele.Value.(*entry) //更新节点的值
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else { //键不存在
		ele := c.ll.PushFront(&entry{key, value}) //队尾添加新节点
		c.cache[key] = ele                        //在字典中添加键与节点的映射关系
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest() //若超过了设定的内存最大值，移除最少访问的节点
	}
}

// 获取缓存条目数(用于测试)
func (c *Cache) Len() int {
	return c.ll.Len()
}
