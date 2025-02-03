package objects

import (
	"sync"
)

// A generic,thread-safe map of objects with auto-inrementing IDs

type SharedCollection[T any] struct {
	objectsMap map[uint64]T
	nextId     uint64
	mapMux     sync.Mutex //互斥锁
}

func NewSharedCollection[T any](capacity ...int) *SharedCollection[T] {
	var newObjMap map[uint64]T

	if len(capacity) > 0 {
		newObjMap = make(map[uint64]T, capacity[0])
	} else {
		newObjMap = make(map[uint64]T)
	}

	return &SharedCollection[T]{
		objectsMap: newObjMap,
		nextId:     1,
	}
}

// Add an object to the map with the given ID (if provided) or the next available ID,
// Returns the ID of the object added. 自增一个ID
func (s *SharedCollection[T]) Add(obj T, id ...uint64) uint64 {
	s.mapMux.Lock()
	defer s.mapMux.Unlock()

	thisId := s.nextId

	if len(id) > 0 {
		thisId = id[0]
	}

	s.objectsMap[thisId] = obj
	s.nextId++

	return thisId
}

func (s *SharedCollection[T]) Remove(id uint64) {
	s.mapMux.Lock()

	defer s.mapMux.Unlock()

	delete(s.objectsMap, id) //从地图上删除
}

// 调用回调函数在遍历地图中
func (s *SharedCollection[T]) ForEach(callback func(uint64, T)) {
	// 创建一个本地副本带锁的
	s.mapMux.Lock()
	localCopy := make(map[uint64]T, len(s.objectsMap))
	for id, obj := range s.objectsMap {
		localCopy[id] = obj
	}
	s.mapMux.Unlock() //解锁

	//无锁状态下调用回调函数
	for id, obj := range localCopy {
		callback(id, obj)
	}
}

// 给定一个对象 判断是否存在 第二个参数则是是否存在
func (s *SharedCollection[T]) Get(id uint64) (T, bool) {
	s.mapMux.Lock()
	defer s.mapMux.Unlock()

	obj, found := s.objectsMap[id]

	return obj, found
}

// 获得地图的长度(不带锁并不精确)
func (s *SharedCollection[T]) Len() int {
	return len(s.objectsMap)
}


