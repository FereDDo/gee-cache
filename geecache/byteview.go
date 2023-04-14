package geecache

// 缓存值的抽象与封装
// 字节视图（只读）
type ByteView struct {
	b []byte //存储真实的缓存值, byte类型能够支持任意数据类型的存储
}

// 返回视图长度
func (v ByteView) Len() int {
	return len(v.b)
}

// 以字节切片的形式返回数据拷贝，防止缓存值被外部程序修改
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// 将数据作为字符串返回，必要时创建拷贝
func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
