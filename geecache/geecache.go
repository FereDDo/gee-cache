package geecache

import (
	"fmt"
	pb "gee-cache/geecache/geecachepb"
	"gee-cache/geecache/singleflight"
	"log"
	"sync"
)

// 定义接口Getter和回调函数
type Getter interface {
	Get(key string) ([]byte, error)
}

// 定义函数类型GetterFunc，并实现Getter接口的方法
type GetterFunc func(key string) ([]byte, error)

// 获取实现Getter接口函数
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// Group负责与用户的交互，并且控制缓存值存储和获取的流程
type Group struct { //缓存的命名空间
	name      string              //组名
	getter    Getter              //缓存未命中时获取源数据的回调函数
	mainCache cache               //初始并发缓存
	peers     PeerPicker          //选择节点
	loader    *singleflight.Group //确保每个key仅响应一次请求
}

var (
	mu     sync.Mutex
	rwmu   sync.RWMutex
	groups = make(map[string]*Group)
)

// 创建Group实例
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

// 根据命名返回已创建Group
func GetGroup(name string) *Group {
	rwmu.RLock() //不涉及任何冲突变量的写操作,使用只读锁
	g := groups[name]
	rwmu.RUnlock()
	return g
}

// 通过key从缓存中获取value
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required")
	}
	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit") //存在，返回缓存值
		return v, nil
	}
	return g.load(key)
}

func (g *Group) load(key string) (value ByteView, err error) {
	viewi, err := g.loader.Do(key, func() (interface{}, error) { //并发情况下对于相同的key，load过程只会调用一次
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok { //选择节点
				if value, err = g.getFromPeer(peer, key); err == nil { //为远程节点
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key) //为本机节点或失败
	})
	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

// 调用用户回调函数获取源数据
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

// 将源数据添加到缓存mainCache中
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

// 注册一个PeerPicker来选择远程节点
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than more")
	}
	g.peers = peers //将PeerPicker接口的HTTPPool注入到Group中
}

// 通过PeerGetter接口的httpGetter访问远程节点，返回缓存值。
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	res := &pb.Response{}
	err := peer.Get(req, res)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: res.Value}, nil
}
