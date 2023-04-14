package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

//实现一致性哈希算法

// 定义Hash函数，采取依赖注入的方式，允许用于替换成自定义的Hash函数，默认为crc32.ChecksumIEEE算法
type Hash func(data []byte) uint32

// 哈希表
type Map struct {
	hash     Hash           //Hash函数
	replicas int            //虚拟节点倍数
	keys     []int          //哈希环
	hashMap  map[int]string //虚拟节点与真实节点之间的映射表
}

// 哈希表构造函数
func New(replicas int, fn Hash) *Map {
	m := &Map{ //可自定义replicas和hash
		hash:     fn,
		replicas: replicas,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// 添加真实节点
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key))) //给虚拟节点编号
			m.keys = append(m.keys, hash)                      //将虚拟节点添加到哈希环上
			m.hashMap[hash] = key                              //在哈希表中添加真实节点和虚拟节点的映射关系
		}
	}
	sort.Ints(m.keys) //对哈希环上的哈希值进行排序
}

// 选择节点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key)))
	//keys为有序数组，使用二分查找
	idx := sort.Search(len(m.keys), func(i int) bool { //找到第一个大于哈希值的项
		return m.keys[i] >= hash
	})
	return m.hashMap[m.keys[idx%len(m.keys)]] //环状结构采取取余
}
